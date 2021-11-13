package model

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/binary"
)

// AESSecurityHandler stores the various data needed
// to crypt/decryt a PDF file.
// It is obtained from user provided passwords and
// data found in Encrypt dictionnary and file trailer.
type AESSecurityHandler struct {
	permissions UserPermissions
}

// NewAESSecurityHandler uses the field in `e` and the provided settings to
// build a `AESSecurityHandler`, which uses AES.
// When crypting a document, an EncryptionStandard shoud then be created and installed on
// the document.
// When decrypting a document, an EncryptionStandard should then be created and compared with
// the one found in the PDF file.
func (e *Encrypt) NewAESSecurityHandler(fileID string, revision uint8, dontEncryptMetadata bool) *AESSecurityHandler {
	return &AESSecurityHandler{
		permissions: e.P,
	}
}

// authOwnerPassword compare the given password to the hash found in a PDF file, returning
// `true` if the owner password is correct, as well as the encryption key
// See - Algorithm 7: Authenticating the owner password
func (ae *AESSecurityHandler) authOwnerPassword(password string, ownerHash, userHash [48]byte, ownerE [32]byte) ([32]byte, bool) {
	opw := []byte(password)
	if len(opw) > 127 {
		opw = opw[:127]
	}

	// Algorithm 3.2a 3.
	b := append(opw, validationSalt(ownerHash[:])...)
	b = append(b, userHash[:]...)
	s := sha256.Sum256(b)

	if !bytes.HasPrefix(ownerHash[:], s[:]) {
		return [32]byte{}, false
	}

	// compute the encryption key
	b = append(opw, keySalt(ownerHash[:])...)
	b = append(b, userHash[:]...)
	key := sha256.Sum256(b)

	cb, _ := aes.NewCipher(key[:])
	var (
		iv     [16]byte
		encKey [32]byte
	)

	mode := cipher.NewCBCDecrypter(cb, iv[:])
	mode.CryptBlocks(encKey[:], ownerE[:])
	return encKey, true
}

// authUserPassword compare the given password to the hash found in a PDF file.
// It returns the encryption key and `true` if the password is correct, or `false`.
// See - Algorithm 6: Authenticating the user password
func (as *AESSecurityHandler) authUserPassword(password string, ownerHash, userHash [48]byte, userE [32]byte) ([32]byte, bool) {
	upw := []byte(password)
	if len(upw) > 127 {
		upw = upw[:127]
	}

	// Algorithm 3.2a 4,
	s := sha256.Sum256(append(upw, validationSalt(userHash[:])...))

	if !bytes.HasPrefix(userHash[:], s[:]) {
		return [32]byte{}, false
	}

	key := sha256.Sum256(append(upw, keySalt(userHash[:])...))

	cb, _ := aes.NewCipher(key[:])
	var (
		iv     [16]byte
		encKey [32]byte
	)

	mode := cipher.NewCBCDecrypter(cb, iv[:])
	mode.CryptBlocks(encKey[:], userE[:])
	return encKey, true
}

// validatePermissions decrypt the Perms and check its consistency against the P entry.
func (as *AESSecurityHandler) validatePermissions(encryptionKey [32]byte, perms [16]byte) bool {
	// Algorithm 3.2a 5.

	cb, _ := aes.NewCipher(encryptionKey[:])

	cb.Decrypt(perms[:], perms[:])

	if string(perms[9:12]) != "adb" {
		return false
	}

	b := binary.LittleEndian.Uint32(perms[:4])
	return int32(b) == int32(as.permissions)
}

func validationSalt(bb []byte) []byte { return bb[32:40] }

func keySalt(bb []byte) []byte { return bb[40:] }

// AuthenticatePasswords compare the given passwords to the hash found in a PDF file, returning
// `true` if one of the password is correct, as well as the encryption key.
func (s *AESSecurityHandler) AuthenticatePasswords(ownerPassword, userPassword string, enc EncryptionStandard) ([]byte, bool) {
	key, ok := s.authOwnerPassword(ownerPassword, enc.O, enc.U, enc.OE)
	if !ok {
		key, ok = s.authUserPassword(userPassword, enc.O, enc.U, enc.UE)
	}

	ok = ok && s.validatePermissions(key, enc.Perms)
	return key[:], ok
}
