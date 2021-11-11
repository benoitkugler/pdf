package model

// adapted from the work of Klemen VODOPIVEC and Kurt Jung

import (
	"bytes"
	"crypto/md5"
	"crypto/rc4"
)

// RC4SecurityHandler stores the various data needed
// to crypt/decryt a PDF file.
// It is obtained from user provided passwords and
// data found in Encrypt dictionnary and file trailer.
type RC4SecurityHandler struct {
	revision           uint8
	specifiedKeyLength int // in bytes, relevant only for revision >= 3

	permissions         UserPermissions
	id                  string
	dontEncryptMetadata bool
}

// NewRC4SecurityHandler uses the field in `e` and the provided settings to
// build a `RC4SecurityHandler`, which uses RC4.
// When crypting a document, an EncryptionStandard shoud then be created and installed on
// the document.
// When decrypting a document, an EncryptionStandard should then be created and compared with
// the one found in the PDF file.
func (e *Encrypt) NewRC4SecurityHandler(fileID string, revision uint8, dontEncryptMetadata bool) *RC4SecurityHandler {
	return &RC4SecurityHandler{
		revision:            revision,
		specifiedKeyLength:  int(e.Length),
		permissions:         e.P,
		id:                  fileID,
		dontEncryptMetadata: dontEncryptMetadata,
	}
}

// Algorithm 2: Computing an encryption key
func (s RC4SecurityHandler) generateEncryptionKey(password string, ownerHash [48]byte) []byte {
	pass := padPassword(password)
	keyLength := s.keyLength()

	buf := append([]byte(nil), pass[:]...)
	buf = append(buf, ownerHash[:32]...)
	buf = append(buf, s.permissions.bytes()...)
	buf = append(buf, s.id...)
	if s.revision >= 4 && s.dontEncryptMetadata {
		buf = append(buf, 0xff, 0xff, 0xff, 0xff)
	}
	sum := md5.Sum(buf)

	if s.revision >= 3 {
		for range [50]int{} {
			sum = md5.Sum(sum[0:keyLength])
		}
	}

	return sum[0:keyLength]
}

// Algorithm 3, steps a) -> d)
func (s *RC4SecurityHandler) generateOwnerEncryptionKey(ownerPassword string) []byte {
	ownerPass := padPassword(ownerPassword)

	keyLength := s.keyLength()

	tmp := md5.Sum(ownerPass[:])
	if s.revision >= 3 {
		for range [50]int{} {
			tmp = md5.Sum(tmp[:])
		}
	}

	return tmp[0:keyLength]
}

// Algorithm 3: Computing the encryption dictionary’s O (owner password) value
func (s *RC4SecurityHandler) generateOwnerHash(userPassword, ownerPassword string) (v [48]byte) {
	firstEncKey := s.generateOwnerEncryptionKey(ownerPassword)

	userPass := padPassword(userPassword)

	c, _ := rc4.NewCipher(firstEncKey)
	c.XORKeyStream(v[:], userPass[:])

	if s.revision >= 3 {
		xor19Times(v[:], firstEncKey)
	}
	return v
}

// Algorithm 4: Computing the encryption dictionary’s U (user password) value (Security handlers of
// revision 2)
// Algorithm 5: Computing the encryption dictionary’s U (user password) value (Security handlers of
// revision 3 or greater)
func (s RC4SecurityHandler) generateUserHash(encryptionKey []byte) (v [48]byte) {
	c, _ := rc4.NewCipher(encryptionKey)
	if s.revision >= 3 {
		buf := padding[:]
		buf = append(buf, s.id...)
		hash := md5.Sum(buf)
		c.XORKeyStream(hash[:], hash[:])
		xor19Times(hash[:], encryptionKey)
		copy(v[0:16], hash[:]) // padding with zeros
	} else {
		c.XORKeyStream(v[:], padding[:])
	}
	return v
}

// AuthUserPassword compare the given password to the hash found in a PDF file.
// It returns the encryption key and `true` if the password is correct, or `false`.
// See - Algorithm 6: Authenticating the user password
func (s *RC4SecurityHandler) AuthUserPassword(password string, ownerHash, userHash [48]byte) ([]byte, bool) {
	encryptionKey := s.generateEncryptionKey(password, ownerHash)
	gotHash := s.generateUserHash(encryptionKey)

	// Quoting the SPEC : comparing on the first 16 bytes in the case of security handlers of revision 3 or greater
	var ok bool
	if s.revision <= 2 {
		ok = bytes.Equal(userHash[:32], gotHash[:32])
	} else {
		ok = bytes.Equal(userHash[:16], gotHash[:16])
	}

	return encryptionKey, ok
}

// AuthOwnerPassword compare the given password to the hash found in a PDF file, returning
// `true` if the owner password is correct, as well as the encryption key.
// See - Algorithm 7: Authenticating the owner password
func (s *RC4SecurityHandler) AuthOwnerPassword(password string, ownerHash, userHash [48]byte) ([]byte, bool) {
	// step a)
	encryptionKey := s.generateOwnerEncryptionKey(password)

	// step b)
	var decryptedPassword [32]byte
	copy(decryptedPassword[:], ownerHash[:32]) // copy to preserve ownerHash
	if s.revision <= 2 {
		c, _ := rc4.NewCipher(encryptionKey)
		c.XORKeyStream(decryptedPassword[:], decryptedPassword[:])
	} else {
		newKey := make([]byte, len(encryptionKey))
		for i := byte(19); i >= 0; i-- {
			for j, b := range encryptionKey { // update `newKey`
				newKey[j] = b ^ i
			}

			c, _ := rc4.NewCipher(newKey)
			c.XORKeyStream(decryptedPassword[:], decryptedPassword[:])
		}
	}

	// step c)
	return s.AuthUserPassword(string(decryptedPassword[:]), ownerHash, userHash)
}
