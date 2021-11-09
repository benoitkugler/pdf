package file

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rc4"
	"errors"

	"github.com/benoitkugler/pdf/model"
)

type encrypt struct {
	key []byte
	aes bool
}

// TODO:
// read the trailer and the Config to build
// the data needed to decode the document
func (ctx *context) setupEncryption() error {
	// encryptDict, err := ctx.resolve(ctx.trailer.encrypt)
	// if err != nil {
	// 	return  err
	// }
	return nil
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
