package model

// adapted from the work of Klemen VODOPIVEC and Kurt Jung

import (
	"crypto/md5"
	"crypto/rc4"
	"encoding/binary"
	"fmt"
)

var padding = [...]byte{
	0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41,
	0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08,
	0x2E, 0x2E, 0x00, 0xB6, 0xD0, 0x68, 0x3E, 0x80,
	0x2F, 0x0C, 0xA9, 0xFE, 0x64, 0x53, 0x69, 0x7A,
}

// UserPermissions is a flag.
// See Table 22 – User access permissions and Table 24 – Public-Key security handler user access permissions
// in the PDF SPEC.
type UserPermissions uint32

const (
	PermissionChangeEncryption UserPermissions = 1 << (2 - 1)  // Permits change of encryption and enables all other permissions.
	PermissionPrint            UserPermissions = 1 << (3 - 1)  // Print the document.
	PermissionModify           UserPermissions = 1 << (4 - 1)  // Modify the contents of the document by operations other than those controlled by bits 6, 9, and 11.
	PermissionCopy             UserPermissions = 1 << (5 - 1)  // Copy or otherwise extract text and graphics from the document
	PermissionAdd              UserPermissions = 1 << (6 - 1)  // Add or modify text annotations, fill in interactive form fields
	PermissionFill             UserPermissions = 1 << (9 - 1)  // Fill in existing interactive form fields
	PermissionExtract          UserPermissions = 1 << (10 - 1) // Extract text and graphics
	PermissionAssemble         UserPermissions = 1 << (11 - 1) // Assemble the document (insert, rotate, or delete pages and create bookmarks or thumbnail images)
	PermissionPrintDigital     UserPermissions = 1 << (12 - 1) // Print the document to a representation from which a faithful digital copy of the PDF content could be generated.
	allRevision3                               = PermissionChangeEncryption | PermissionPrint | PermissionCopy | PermissionFill | PermissionExtract | PermissionAssemble | PermissionPrintDigital
)

// write u as 4 bytes, low-order byte first.
func (u UserPermissions) bytes() []byte {
	var out [4]byte
	binary.LittleEndian.PutUint32(out[:], uint32(u))
	return out[:]
}

// return true if `u` has any of the flags “Security handlers of revision 3 or greater”
// set to 0
func (u UserPermissions) isRevision3() bool {
	b := (u & allRevision3) == allRevision3 // all flags rev 3 are set
	return !b
}

type EncryptionStandard struct {
	R uint8 // 2 ,3 or 4
	O [32]byte
	U [32]byte
	// optional, default value is false
	// written in PDF under the key /EncryptMetadata
	DontEncryptMetadata bool

	// needed to encrypt, but not written in the PDF
	encryptionKey []byte
}

// SetStandardEncryptionHandler create a Standard security handler
// and install it on the document.
// The field V and P of the encrypt dict must be setup previously.
// `userPassword` and `ownerPassword` are used to generate the encryption keys
// and will be needed to decrypt the document.
func (t *Trailer) SetStandardEncryptionHandler(userPassword, ownerPassword string, encryptMetadata bool) {
	enc := &t.Encrypt
	enc.Filter = "Standard"
	enc.SubFilter = ""
	var out EncryptionStandard
	out.DontEncryptMetadata = !encryptMetadata
	if enc.V < 2 && !enc.P.isRevision3() {
		out.R = 2
	} else if enc.V == 2 || enc.V == 3 || enc.P.isRevision3() {
		out.R = 3
	} else if enc.V == KeySecurityHandler {
		out.R = 4
	}

	keyLength := 5
	if out.R >= 3 && enc.Length != 0 {
		keyLength = int(enc.Length)
	}

	var userPass, ownerPass [32]byte
	copy(userPass[:], append([]byte(userPassword), padding[:]...)[0:32])
	copy(ownerPass[:], append([]byte(ownerPassword), padding[:]...)[0:32])

	out.O = generateOwnerHash(out.R, keyLength, userPass, ownerPass)
	buf := append([]byte(nil), userPass[:]...)
	buf = append(buf, out.O[:]...)
	buf = append(buf, enc.P.bytes()...)
	buf = append(buf, t.ID[0]...)
	if out.R >= 4 && !encryptMetadata {
		buf = append(buf, 0xff, 0xff, 0xff, 0xff)
	}
	sum := md5.Sum(buf)

	if out.R >= 3 {
		for range [50]int{} {
			sum = md5.Sum(sum[0:keyLength])
		}
	}
	out.encryptionKey = sum[0:keyLength]
	out.U = t.generateUserHash(out.R, out.encryptionKey)

	enc.EncryptionHandler = out
}

func (e EncryptionStandard) encryptionAddFields() string {
	return fmt.Sprintf("/R %d /O %s /U %s /EncryptMetadata %v",
		e.R, escapeFormatByteString(string(e.O[:])),
		escapeFormatByteString(string(e.U[:])), !e.DontEncryptMetadata)
}

func (e EncryptionStandard) Clone() EncryptionHandler {
	out := e
	out.encryptionKey = append([]byte(nil), e.encryptionKey...)
	return out
}

func (e EncryptionStandard) isSetup() bool {
	return len(e.encryptionKey) >= 0
}

// Crypt encrypt in-place the given `data` using its object number,
// with the RC4 algorithm.
func (p EncryptionStandard) Crypt(n Reference, data []byte) ([]byte, error) {
	out := make([]byte, len(data))
	rc4cipher, _ := rc4.NewCipher(objectEncrytionKey(p.encryptionKey, n, false))
	rc4cipher.XORKeyStream(out, data)
	return out, nil
}

func objectEncrytionKey(baseKey []byte, n Reference, aes bool) []byte {
	var nbuf [4]byte
	binary.LittleEndian.PutUint32(nbuf[:], uint32(n))
	b := append(baseKey, nbuf[0], nbuf[1], nbuf[2], 0, 0) // copy and padding (generation number is 0)
	if aes {
		b = append(b, 0x73, 0x41, 0x6C, 0x54) // append sAlT
	}
	s := md5.Sum(b)
	size := len(baseKey) + 5
	if size > 16 {
		size = 16
	}
	return s[0:size]
}

func generateOwnerHash(revision uint8, keyLength int, userPass, ownerPass [32]byte) (v [32]byte) {
	tmp := md5.Sum(ownerPass[:])
	if revision >= 3 {
		for range [50]int{} {
			tmp = md5.Sum(tmp[:])
		}
	}
	firstEncKey := tmp[0:keyLength]
	c, _ := rc4.NewCipher(firstEncKey)
	c.XORKeyStream(v[:], userPass[:])

	if revision >= 3 {
		xor19(v[:], firstEncKey)
	}
	return v
}

// write into initial
func xor19(initial []byte, startEncKey []byte) {
	for i := 1; i <= 19; i++ {
		newKey := append([]byte(nil), startEncKey...) // copy to preserve startEncKey
		for j, b := range newKey {
			newKey[j] = b ^ byte(i)
		}
		c, _ := rc4.NewCipher(newKey)
		c.XORKeyStream(initial, initial)
	}
}

func (t Trailer) generateUserHash(revision uint8, encryptionKey []byte) (v [32]byte) {
	c, _ := rc4.NewCipher(encryptionKey)
	if revision >= 3 {
		buf := padding[:]
		buf = append(buf, t.ID[0]...)
		hash := md5.Sum(buf)
		c.XORKeyStream(hash[:], hash[:])
		xor19(hash[:], encryptionKey)
		copy(v[0:16], hash[:]) // padding with zeros
	} else {
		c.XORKeyStream(v[:], padding[:])
	}
	return v
}

// ------------------------------------------------------------------------------------

// Crypt is not supported for the PublicKey security handler
// Thus, this function return the plain data.
func (e EncryptionPublicKey) Crypt(n Reference, data []byte) ([]byte, error) {
	return data, nil
}

func (e EncryptionPublicKey) isSetup() bool { return false }

// func cryptAes(objectKey, data []byte) ([]byte, error) {
// 	// pad data to aes.Blocksize
// 	l := len(data) % aes.BlockSize
// 	var c byte = 0x10
// 	if l > 0 {
// 		c = byte(aes.BlockSize - l)
// 	}
// 	data = append(data, bytes.Repeat([]byte{c}, aes.BlockSize-l)...)
// 	// now, len(data) >= 16 and len(data)%16 == 0

// 	block := make([]byte, aes.BlockSize+len(data)) // room for 16 random bytes
// 	iv := block[:aes.BlockSize]

// 	_, err := io.ReadFull(rand.Reader, iv)
// 	if err != nil {
// 		return nil, err
// 	}

// 	cb, err := aes.NewCipher(objectKey)
// 	if err != nil {
// 		return nil, err
// 	}

// 	mode := cipher.NewCBCEncrypter(cb, iv)
// 	mode.CryptBlocks(block[aes.BlockSize:], data)

// 	return block, nil
// }

// func (s EncryptionPublicKey) generateEncryptionKey(keyLength uint8, cryptMetadata bool) ([]byte, error) {
// 	data := make([]byte, 20) // a)
// 	_, err := io.ReadFull(rand.Reader, data)
// 	if err != nil {
// 		return nil, err
// 	}

// 	for _, rec := range s { // b)
// 		data = append(data, rec...)
// 	}

// 	if !cryptMetadata { // c)
// 		data = append(data, 0xff, 0xff, 0xff, 0xff)
// 	}
// 	sum := sha1.Sum(data)
// 	return sum[0:keyLength], nil
// }
