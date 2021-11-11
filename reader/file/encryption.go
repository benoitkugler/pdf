package file

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rc4"
	"errors"
	"fmt"

	"github.com/benoitkugler/pdf/model"
)

type encrypt struct {
	enc model.Encrypt // found in the PDF file
	ID  [2]string     // found in the PDF file

	key                    []byte
	aesStrings, aesStreams bool
}

// Read the trailer and the Config to build
// the data needed to decode the document
// In particular, authenticates the user provided passwords.
func (ctx *context) setupEncryption() (err error) {
	if ctx.trailer.encrypt == nil { // not encrypted
		return nil
	}

	var info encrypt

	info.enc, err = ctx.processEncryptDict()
	if err != nil {
		return err
	}

	if info.enc.StmF != "" && info.enc.StmF != "Identity" {
		d, ok := info.enc.CF[info.enc.StmF]
		if !ok {
			return fmt.Errorf("missing entry for StmF %s in CF encrypt dict", info.enc.StmF)
		}

		info.aesStreams, err = supportedCFEntry(d)
		if err != nil {
			return err
		}
	}

	if info.enc.StrF != "" && info.enc.StrF != "Identity" {
		d, ok := info.enc.CF[info.enc.StrF]
		if !ok {
			return fmt.Errorf("missing entry for StrF %s in CF encrypt dict", info.enc.StrF)
		}

		info.aesStrings, err = supportedCFEntry(d)
		if err != nil {
			return err
		}
	}

	if info.enc.Filter != "Standard" {
		return fmt.Errorf("unsupported encryption handler %s", info.enc.Filter)
	}
	if info.enc.SubFilter != "" {
		return fmt.Errorf("unsupported encryption handler SubFilter %s", info.enc.SubFilter)
	}

	if len(ctx.trailer.id) != 2 {
		return fmt.Errorf("invalid ID entry for trailer, expected 2-length string array, got %v", ctx.trailer.id)
	}
	info.ID[0], _ = IsString(ctx.trailer.id[0])
	info.ID[1], _ = IsString(ctx.trailer.id[1])

	e, _ := info.enc.EncryptionHandler.(model.EncryptionStandard)
	if e.R == 5 {
		// TODO:
	} else {
		sh := info.enc.NewRC4SecurityHandler(info.ID[0], e.R, e.DontEncryptMetadata)

		// first try with owner
		var ok bool
		info.key, ok = sh.AuthOwnerPassword(ctx.Configuration.Password, e.O, e.U)
		if !ok {
			info.key, ok = sh.AuthUserPassword(ctx.Configuration.Password, e.O, e.U)
			if !ok {
				return fmt.Errorf("incorrect password: %s", ctx.Configuration.Password)
			}
		}
	}

	ctx.enc = &info

	return nil
}

// supportedCFEntry returns true if AES should be used,
// or an error is the fields are invalid
func supportedCFEntry(d model.CrypFilter) (bool, error) {
	cfm := d.CFM
	if cfm != "" && cfm != "V2" && cfm != "AESV2" && cfm != "AESV3" {
		return false, fmt.Errorf("invalid CFM entry %s", cfm)
	}

	// don't check for d.AuthEvent since :
	// If this filter is used as the value of StrF or StmF in the encryption
	// dictionary (see Table 20), the conforming reader shall ignore this key
	// and behave as if the value is DocOpen.

	if l := d.Length; l != 0 && (l < 5 || l > 16) && l != 32 {
		return false, fmt.Errorf("invalid Length entry %d", l)
	}

	return cfm == "AESV2" || cfm == "AESV3", nil
}

// the returned key is 5 to 16 byte long
func decryptKey(key []byte, objNumber, generationNumber int, useAES bool) []byte {
	b := append(key,
		byte(objNumber), byte(objNumber>>8), byte(objNumber>>16),
		byte(generationNumber), byte(generationNumber>>8),
	)

	if useAES {
		b = append(b, "sAlT"...)
	}

	dk := md5.Sum(b)

	l := len(key) + 5
	if l < 16 {
		return dk[:l]
	}

	return dk[:]
}

// content may be overwritten
func decryptBytes(content []byte, ref model.ObjIndirectRef, useAES bool, key []byte) ([]byte, error) {
	key = decryptKey(key, ref.ObjectNumber, ref.GenerationNumber, useAES)
	if useAES {
		return decryptAESBytes(content, key)
	}

	return decryptRC4Bytes(content, key)
}

func decryptRC4Bytes(buf, key []byte) ([]byte, error) {
	c, _ := rc4.NewCipher(key)
	c.XORKeyStream(buf, buf)
	return buf, nil
}

func decryptAESBytes(b, key []byte) ([]byte, error) {
	if len(b) < aes.BlockSize {
		return nil, errors.New("decryptAESBytes: Ciphertext too short")
	}

	if len(b)%aes.BlockSize > 0 {
		return nil, errors.New("decryptAESBytes: Ciphertext not a multiple of block size")
	}

	cb, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	iv := make([]byte, aes.BlockSize)
	copy(iv, b[:aes.BlockSize])

	data := b[aes.BlockSize:]
	mode := cipher.NewCBCDecrypter(cb, iv)
	mode.CryptBlocks(data, data)

	// Remove padding.
	// Note: For some reason not all AES ciphertexts are padded.
	if len(data) > 0 && data[len(data)-1] <= 0x10 {
		e := len(data) - int(data[len(data)-1])
		data = data[:e]
	}

	return data, nil
}

// recursively walk through the object and decrypt strings and streams
func (enc *encrypt) decryptObject(o model.Object, contextRef model.ObjIndirectRef) (model.Object, error) {
	var err error
	switch oT := o.(type) {
	case model.ObjHexLiteral: // do the actual decryption
		decrypted, err := decryptBytes([]byte(oT), contextRef, enc.aesStrings, enc.key)
		if err != nil {
			return nil, err
		}
		o = model.ObjHexLiteral(string(decrypted))
	case model.ObjStringLiteral: // do the actual decryption
		decrypted, err := decryptBytes([]byte(oT), contextRef, enc.aesStrings, enc.key)
		if err != nil {
			return nil, err
		}
		o = model.ObjStringLiteral(string(decrypted))
	case model.ObjDict: // recurse
		for k, v := range oT {
			oT[k], err = enc.decryptObject(v, contextRef)
			if err != nil {
				return nil, err
			}
		}
	case model.ObjStream: // recurse
		argsO, err := enc.decryptObject(oT.Args, contextRef)
		if err != nil {
			return nil, err
		}
		content, err := enc.decryptStream(oT.Content, contextRef)
		if err != nil {
			return nil, err
		}
		// correct the Length args so that it matches the decrypted content
		args := argsO.(model.ObjDict)
		args["Length"] = model.ObjInt(len(content))
		o = model.ObjStream{Args: args, Content: content}
	case model.ObjArray: // recurse
		for i, v := range oT {
			oT[i], err = enc.decryptObject(v, contextRef)
			if err != nil {
				return nil, err
			}
		}
	}
	return o, nil
}

func (enc *encrypt) decryptStream(content []byte, ref model.ObjIndirectRef) ([]byte, error) {
	return decryptBytes(content, ref, enc.aesStreams, enc.key)
}

// used only for the encrypt dict, where all object should probably be direct
func (ctx *context) res(obj model.Object) model.Object {
	out, _ := ctx.resolve(obj)
	return out
}

func (ctx *context) processEncryptDict() (model.Encrypt, error) {
	var out model.Encrypt

	encryptO, err := ctx.resolve(ctx.trailer.encrypt)
	if err != nil {
		return out, err
	}
	d, _ := encryptO.(model.ObjDict)

	out.Filter, _ = ctx.res(d["Filter"]).(model.ObjName)
	out.SubFilter, _ = ctx.res(d["SubFilter"]).(model.ObjName)

	v, _ := ctx.res(d["V"]).(model.ObjInt)
	out.V = model.EncryptionAlgorithm(v)

	length, _ := ctx.res(d["Length"]).(model.ObjInt)
	if length%8 != 0 {
		return out, fmt.Errorf("field Length must be a multiple of 8")
	}
	// we accept up to 256, which is the limit of the RC4 cipher
	if length < 0 || length > 256 {
		return out, fmt.Errorf("field Length must be between 0 and 256, got %d", length)
	}
	out.Length = uint8(length / 8)

	cf, _ := ctx.res(d["CF"]).(model.ObjDict)
	out.CF = make(map[model.ObjName]model.CrypFilter, len(cf))
	for name, c := range cf {
		out.CF[model.ObjName(name)] = ctx.processCryptFilter(c)
	}
	out.StmF, _ = ctx.res(d["StmF"]).(model.ObjName)
	out.StrF, _ = ctx.res(d["StrF"]).(model.ObjName)
	out.EFF, _ = ctx.res(d["EFF"]).(model.ObjName)

	p, _ := ctx.res(d["P"]).(model.ObjInt)
	out.P = model.UserPermissions(p)

	// subtypes
	if out.Filter == "Standard" {
		out.EncryptionHandler, err = ctx.processStandardSecurityHandler(d)
		if err != nil {
			return out, err
		}
	} else {
		out.EncryptionHandler = ctx.processPublicKeySecurityHandler(d)
	}

	return out, nil
}

func (ctx *context) processStandardSecurityHandler(dict model.ObjDict) (model.EncryptionStandard, error) {
	var out model.EncryptionStandard
	r_, _ := ctx.res(dict["R"]).(model.ObjInt)
	out.R = uint8(r_)

	hashLength := 32
	if out.R == 5 { // AES
		hashLength = 48
	}
	o, _ := IsString(ctx.res(dict["O"]))
	if len(o) != hashLength {
		return out, fmt.Errorf("expected %d-length byte string for entry O, got %v", hashLength, o)
	}
	copy(out.O[:], o)

	u, _ := IsString(ctx.res(dict["U"]))
	if len(u) != hashLength {
		return out, fmt.Errorf("expected %d-length byte string for entry U, got %v", hashLength, u)
	}
	copy(out.U[:], u)

	if out.R == 5 {
		ue, _ := IsString(ctx.res(dict["UE"]))
		if len(ue) != 32 {
			return out, fmt.Errorf("expected %d-length byte string for entry UE, got %v", 32, ue)
		}
		copy(out.UE[:], u)

		oe, _ := IsString(ctx.res(dict["OE"]))
		if len(oe) != 32 {
			return out, fmt.Errorf("expected %d-length byte string for entry OE, got %v", 32, oe)
		}
		copy(out.OE[:], u)
	}

	if meta, ok := ctx.res(dict["EncryptMetadata"]).(model.ObjBool); ok {
		out.DontEncryptMetadata = !bool(meta)
	}
	return out, nil
}

func (ctx *context) processPublicKeySecurityHandler(dict model.ObjDict) model.EncryptionPublicKey {
	rec, _ := ctx.res(dict["Recipients"]).(model.ObjArray)
	out := make(model.EncryptionPublicKey, len(rec))
	for i, re := range rec {
		out[i], _ = IsString(ctx.res(re))
	}
	return out
}

func (ctx *context) processCryptFilter(crypt model.Object) model.CrypFilter {
	cryptDict, _ := ctx.res(crypt).(model.ObjDict)
	var out model.CrypFilter
	out.CFM, _ = ctx.res(cryptDict["CFM"]).(model.ObjName)
	out.AuthEvent, _ = ctx.res(cryptDict["AuthEvent"]).(model.ObjName)
	l, _ := ctx.res(cryptDict["AuthEvent"]).(model.ObjInt)
	out.Length = int(l)
	recipients := ctx.res(cryptDict["Recipients"])
	if rec, ok := IsString(recipients); ok {
		out.Recipients = []string{rec}
	} else if ar, ok := recipients.(model.ObjArray); ok {
		out.Recipients = make([]string, len(ar))
		for i, re := range ar {
			out.Recipients[i], _ = IsString(ctx.res(re))
		}
	}
	if enc, ok := ctx.res(cryptDict["EncryptMetadata"]).(model.ObjBool); ok {
		out.DontEncryptMetadata = !bool(enc)
	}
	return out
}
