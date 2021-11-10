package file

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rc4"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/benoitkugler/pdf/model"
)

type encrypt struct {
	enc model.Encrypt // found in the PDF file

	key []byte
	aes bool
}

// read the trailer and the Config to build
// the data needed to decode the document
func (ctx *context) setupEncryption() (err error) {
	if ctx.trailer.encrypt == nil { // not encrypted
		return nil
	}

	var info encrypt

	info.enc, err = ctx.processEncryptDict()
	if err != nil {
		return err
	}

	if info.enc.StmF != "" && info.enc.StmF != "Identiy" {
		d, ok := info.enc.CF[info.enc.StmF]
		if !ok {
			return fmt.Errorf("missing entry for StmF %s in CF encrypt dict", info.enc.StmF)
		}

		info.aes, err = supportedCFEntry(d)
		if err != nil {
			return err
		}
	}
	fmt.Println(encrypt)
	return nil
}

func validationSalt(bb []byte) []byte {
	return bb[32:40]
}

func keySalt(bb []byte) []byte {
	return bb[40:]
}

func validateOwnerPasswordAES256(ctx *Context) (ok bool, err error) {
	if len(ctx.OwnerPW) == 0 {
		return false, nil
	}

	// TODO Process PW with SASLPrep profile (RFC 4013) of stringprep (RFC 3454).
	opw := []byte(ctx.OwnerPW)
	if len(opw) > 127 {
		opw = opw[:127]
	}
	// fmt.Printf("opw <%s> isValidUTF8String: %t\n", opw, utf8.Valid(opw))

	// Algorithm 3.2a 3.
	b := append(opw, validationSalt(ctx.E.O)...)
	b = append(b, ctx.E.U...)
	s := sha256.Sum256(b)

	if !bytes.HasPrefix(ctx.E.O, s[:]) {
		return false, nil
	}

	b = append(opw, keySalt(ctx.E.O)...)
	b = append(b, ctx.E.U...)
	key := sha256.Sum256(b)

	cb, err := aes.NewCipher(key[:])
	if err != nil {
		return false, err
	}

	iv := make([]byte, 16)
	ctx.EncKey = make([]byte, 32)

	mode := cipher.NewCBCDecrypter(cb, iv)
	mode.CryptBlocks(ctx.EncKey, ctx.E.OE)

	return true, nil
}

func validateUserPasswordAES256(ctx *Context) (ok bool, err error) {
	// TODO Process PW with SASLPrep profile (RFC 4013) of stringprep (RFC 3454).
	upw := []byte(ctx.UserPW)
	if len(upw) > 127 {
		upw = upw[:127]
	}
	// fmt.Printf("upw <%s> isValidUTF8String: %t\n", upw, utf8.Valid(upw))

	// Algorithm 3.2a 4,
	s := sha256.Sum256(append(upw, validationSalt(ctx.E.U)...))

	if !bytes.HasPrefix(ctx.E.U, s[:]) {
		return false, nil
	}

	key := sha256.Sum256(append(upw, keySalt(ctx.E.U)...))

	cb, err := aes.NewCipher(key[:])
	if err != nil {
		return false, err
	}

	iv := make([]byte, 16)
	ctx.EncKey = make([]byte, 32)

	mode := cipher.NewCBCDecrypter(cb, iv)
	mode.CryptBlocks(ctx.EncKey, ctx.E.UE)

	return true, nil
}

// ValidateOwnerPassword validates the owner password aka change permissions password.
func (enc encrypt) validateOwnerPassword() (ok bool, err error) {}

func validateOwnerPasswordRC4(enc model.EncryptionStandard) (ok bool, err error) {
	e := ctx.E

	// The PW string is generated from OS codepage characters by first converting the string to
	// PDFDocEncoding. If input is Unicode, first convert to a codepage encoding , and then to
	// PDFDocEncoding for backward compatibility.

	ownerpw := ctx.OwnerPW
	userpw := ctx.UserPW

	// 7a: Alg.3 p62 a-d
	key := key(ownerpw, userpw, e.R, e.L)

	// 7b
	upw := make([]byte, len(e.O))
	copy(upw, e.O)

	if enc.R <= 2 {
		c, err := rc4.NewCipher(key)
		if err != nil {
			return false, err
		}
		c.XORKeyStream(upw, upw)
	} else {
		keynew := make([]byte, len(key))
		for i := 19; i >= 0; i-- {
			for j, b := range key {
				keynew[j] = b ^ byte(i)
			}

			c, err := rc4.NewCipher(keynew)
			if err != nil {
				return false, err
			}

			c.XORKeyStream(upw, upw)
		}
	}

	// Save user pw
	upws := ctx.UserPW

	ctx.UserPW = string(upw)
	ok, err = validateUserPassword(ctx)

	// Restore user pw
	ctx.UserPW = upws

	return ok, err
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

func (enc encrypt) decryptKey(objNumber, generationNumber int) []byte {
	b := append(enc.key,
		byte(objNumber), byte(objNumber>>8), byte(objNumber>>16),
		byte(generationNumber), byte(generationNumber>>8),
	)

	if enc.aes {
		b = append(b, "sAlT"...)
	}

	dk := md5.Sum(b)

	l := len(enc.key) + 5
	if l < 16 {
		return dk[:l]
	}

	return dk[:]
}

func (ctx *context) decryptStream(content []byte, ref model.ObjIndirectRef) ([]byte, error) {
	key := ctx.enc.decryptKey(ref.ObjectNumber, ref.GenerationNumber)

	if ctx.enc.aes {
		return decryptAESBytes(content, key)
	}

	return decryptRC4Bytes(content, key)
}

func decryptRC4Bytes(buf, key []byte) ([]byte, error) {
	c, err := rc4.NewCipher(key)
	if err != nil {
		return nil, err
	}

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

// recursively walk through the object and decrypt strings
// streams are not handled
func (ctx *context) decryptObject(o model.Object) (model.Object, error) {
	var err error
	switch o := o.(type) {
	case model.ObjHexLiteral: // do the actual decryption
		// TODO:
	case model.ObjStringLiteral: // do the actual decryption
		// TODO:
	case model.ObjDict: // recurse
		for k, v := range o {
			o[k], err = ctx.decryptObject(v)
			if err != nil {
				return nil, err
			}
		}
	case model.ObjArray: // recurse
		for i, v := range o {
			o[i], err = ctx.decryptObject(v)
			if err != nil {
				return nil, err
			}
		}
	}
	return o, nil
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

	o, _ := IsString(ctx.res(dict["O"]))
	if len(o) != 32 {
		return out, fmt.Errorf("expected 32-length byte string for entry O, got %v", o)
	}
	for i := 0; i < len(o); i++ {
		out.O[i] = o[i]
	}

	u, _ := IsString(ctx.res(dict["U"]))
	if len(u) != 32 {
		return out, fmt.Errorf("expected 32-length byte string for entry U, got %v", u)
	}
	for i := 0; i < len(u); i++ {
		out.U[i] = u[i]
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
