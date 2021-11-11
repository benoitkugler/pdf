package model

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
)

// AESSecurityHandler stores the various data needed
// to crypt/decryt a PDF file.
// It is obtained from user provided passwords and
// data found in Encrypt dictionnary and file trailer.
type AESSecurityHandler struct {
	revision           uint8
	specifiedKeyLength int // in bytes, relevant only for revision >= 3

	permissions         UserPermissions
	id                  string
	dontEncryptMetadata bool
}

// NewAESSecurityHandler uses the field in `e` and the provided settings to
// build a `AESSecurityHandler`, which uses AES.
// When crypting a document, an EncryptionStandard shoud then be created and installed on
// the document.
// When decrypting a document, an EncryptionStandard should then be created and compared with
// the one found in the PDF file.
func (e *Encrypt) NewAESSecurityHandler(fileID string, revision uint8, dontEncryptMetadata bool) *AESSecurityHandler {
	return &AESSecurityHandler{
		revision:            revision,
		specifiedKeyLength:  int(e.Length),
		permissions:         e.P,
		id:                  fileID,
		dontEncryptMetadata: dontEncryptMetadata,
	}
}

// // Algorithm 3: Computing the encryption dictionary’s O (owner password) value
// func (s *RC4SecurityHandler) generateOwnerHash(userPassword, ownerPassword string) (v [48]byte) {
// 	firstEncKey := s.generateOwnerEncryptionKey(ownerPassword)

// 	userPass := padPassword(userPassword)

// 	c, _ := rc4.NewCipher(firstEncKey)
// 	c.XORKeyStream(v[:], userPass[:])

// 	if s.revision >= 3 {
// 		xor19Times(v[:], firstEncKey)
// 	}
// 	return v
// }

// // Algorithm 4: Computing the encryption dictionary’s U (user password) value (Security handlers of
// // revision 2)
// // Algorithm 5: Computing the encryption dictionary’s U (user password) value (Security handlers of
// // revision 3 or greater)
// func (s RC4SecurityHandler) generateUserHash(encryptionKey []byte) (v [32]byte) {
// 	c, _ := rc4.NewCipher(encryptionKey)
// 	if s.revision >= 3 {
// 		buf := padding[:]
// 		buf = append(buf, s.id...)
// 		hash := md5.Sum(buf)
// 		c.XORKeyStream(hash[:], hash[:])
// 		xor19Times(hash[:], encryptionKey)
// 		copy(v[0:16], hash[:]) // padding with zeros
// 	} else {
// 		c.XORKeyStream(v[:], padding[:])
// 	}
// 	return v
// }

// AuthUserPassword compare the given password to the hash found in a PDF file.
// It returns the encryption key and `true` if the password is correct, or `false`.
// See - Algorithm 6: Authenticating the user password
func (as *AESSecurityHandler) AuthUserPassword(password string, userHash, ownerHash [32]byte, userE []byte) ([]byte, bool) {
	upw := []byte(password)
	if len(upw) > 127 {
		upw = upw[:127]
	}

	// Algorithm 3.2a 4,
	s := sha256.Sum256(append(upw, validationSalt(userHash[:])...))

	if !bytes.HasPrefix(userHash[:], s[:]) {
		return nil, false
	}

	key := sha256.Sum256(append(upw, keySalt(userHash[:])...))

	cb, _ := aes.NewCipher(key[:])
	var (
		iv     [16]byte
		encKey [32]byte
	)

	mode := cipher.NewCBCDecrypter(cb, iv[:])
	mode.CryptBlocks(encKey[:], userE)
	return encKey[:], true
}

func validationSalt(bb []byte) []byte { return bb[32:40] }

func keySalt(bb []byte) []byte { return bb[40:] }

// AuthOwnerPassword compare the given password to the hash found in a PDF file, returning
// `true` if the owner password is correct.
// See - Algorithm 7: Authenticating the owner password
func (ae *AESSecurityHandler) AuthOwnerPassword(password string, userHash, ownerHash [48]byte, ownerE []byte) bool {
	opw := []byte(password)
	if len(opw) > 127 {
		opw = opw[:127]
	}

	// Algorithm 3.2a 3.
	b := append(opw, validationSalt(ownerHash[:])...)
	b = append(b, userHash[:]...)
	s := sha256.Sum256(b)

	if !bytes.HasPrefix(ownerHash[:], s[:]) {
		return false
	}

	// b = append(opw, keySalt(ownerHash[:])...)
	// b = append(b, userHash[:]...)
	// key := sha256.Sum256(b)

	// cb, _ := aes.NewCipher(key[:])
	// var (
	// 	iv     [16]byte
	// 	encKey [32]byte
	// )

	// mode := cipher.NewCBCDecrypter(cb, iv[:])
	// mode.CryptBlocks(encKey[:], ownerE)
	// // TODO:

	return true
}
