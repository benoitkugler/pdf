package model

// adapted from the work of Klemen VODOPIVEC and Kurt Jung

import (
	"crypto/md5"
	"crypto/rc4"
	"encoding/binary"
	"fmt"
	"strings"
)

var padding = [...]byte{
	0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41,
	0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08,
	0x2E, 0x2E, 0x00, 0xB6, 0xD0, 0x68, 0x3E, 0x80,
	0x2F, 0x0C, 0xA9, 0xFE, 0x64, 0x53, 0x69, 0x7A,
}

type SecuriyHandler interface {
	// compare the given passwords against the hash found in a PDF file,
	// and return the encryption key and true if they are correct, or false
	AuthenticatePasswords(ownerPassword, userPassword string, enc EncryptionStandard) ([]byte, bool)
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

// EncryptionAlgorithm is a code specifying the algorithm to be used in encrypting and
// decrypting the document
type EncryptionAlgorithm uint8

const (
	_ EncryptionAlgorithm = iota
	EaRC440
	EaRC4Ext // encryption key with length greater than 40
	_
	EaRC4Custom
	EaAES // AES is used for all content
)

// Encrypt stores the encryption-related information
// It will be filled when reading an existing PDF document.
// Note that to encrypt a document when writting it,
// a call to `Document.UseStandardEncryptionHandler` is needed
// (partly because password are needed, which are not contained in the PDF).
// Also note that encryption with a public key is not supported.
type Encrypt struct {
	EncryptionHandler EncryptionHandler
	Filter            Name
	SubFilter         Name
	V                 EncryptionAlgorithm

	// Length in bytes, from 5 to 16, optional, default to 5.
	// It is written in pdf as bit length.
	Length uint8
	CF     map[Name]CrypFilter // optional
	StmF   Name                // optional
	StrF   Name                // optional
	EFF    Name                // optional
	P      UserPermissions
}

func (e Encrypt) Clone() Encrypt {
	out := e
	if e.EncryptionHandler != nil {
		out.EncryptionHandler = e.EncryptionHandler.Clone()
	}
	if e.CF != nil { // preserve reflet.DeepEqual
		out.CF = make(map[Name]CrypFilter, len(e.CF))
		for k, v := range e.CF {
			out.CF[k] = v.Clone()
		}
	}
	return out
}

func (e Encrypt) pdfString() string {
	b := newBuffer()
	b.line("<<")
	b.fmt("/Filter %s /V %d /P %d", e.Filter, e.V, e.P)
	if e.Length != 0 {
		b.fmt("/Length %d", e.Length*8)
	}
	if e.SubFilter != "" {
		b.fmt("/SubFilter %s", e.SubFilter)
	}
	if e.EncryptionHandler != nil {
		b.WriteString(e.EncryptionHandler.encryptionAddFields() + "\n")
	}
	if e.StmF != "" {
		b.fmt("/StmF %s", e.StmF)
	}
	if e.StrF != "" {
		b.fmt("/StrF %s", e.StrF)
	}
	if e.EFF != "" {
		b.fmt("/EFF %s", e.EFF)
	}
	if e.CF != nil {
		b.fmt("/CF <<")
		for n, v := range e.CF {
			b.fmt("%s %s ", n, v.pdfString(true))
		}
		b.line(">>")
	}
	b.WriteString(">>")
	return b.String()
}

type CrypFilter struct {
	CFM       Name // optional
	AuthEvent Name // optional
	Length    int  // optional

	// byte strings, required for public-key security handlers
	// for Crypt filter decode parameter dictionary,
	// a one element array is written in PDF directly as a string
	Recipients []string
	// optional, default to false
	// written in PDF under the key /EncryptMetadata
	DontEncryptMetadata bool
}

func (c CrypFilter) pdfString(fromCrypt bool) string {
	out := "<<"
	if c.CFM != "" {
		out += "/CFM " + c.CFM.String()
	}
	if c.AuthEvent != "" {
		out += "/AuthEvent " + c.AuthEvent.String()
	}
	if c.Length != 0 {
		out += fmt.Sprintf("/Length %d", c.Length)
	}
	if fromCrypt && len(c.Recipients) == 1 {
		out += "/Recipients " + EscapeByteString([]byte(c.Recipients[0]))
	}
	out += fmt.Sprintf("/EncryptMetadata %v>>", !c.DontEncryptMetadata)
	return out
}

// Clone returns a deep copy
func (c CrypFilter) Clone() CrypFilter {
	out := c
	out.Recipients = append([]string(nil), c.Recipients...)
	return out
}

// EncryptionHandler is either EncryptionStandard or EncryptionPublicKey
type EncryptionHandler interface {
	encryptionAddFields() string
	// Clone returns a deep copy, preserving the concrete type.
	Clone() EncryptionHandler
	// crypt transform the incoming `data`, using `n`
	// as the object number of its context, and return the encrypted bytes.
	crypt(n Reference, data []byte) ([]byte, error)
}

// EncryptionPublicKey is written in PDF under the /Recipients key.
type EncryptionPublicKey []string

func (e EncryptionPublicKey) encryptionAddFields() string {
	chunks := make([]string, len(e))
	for i, s := range e {
		chunks[i] = EscapeByteString([]byte(s))
	}
	return fmt.Sprintf("/Recipients [%s]", strings.Join(chunks, " "))
}

func (e EncryptionPublicKey) Clone() EncryptionHandler {
	return append(EncryptionPublicKey(nil), e...)
}

type EncryptionStandard struct {
	R uint8    // 2, 3, 4 or 5
	O [48]byte // only the first 32 bytes are used when R != 5
	U [48]byte // only the first 32 bytes are used when R != 5

	// optional, default value is false
	// written in PDF under the key /EncryptMetadata
	DontEncryptMetadata bool

	UE, OE [32]byte // used for AES encryption
	Perms  [16]byte // used for AES encryption

	// needed to encrypt, but not written in the PDF
	encryptionKey []byte
}

// UseStandardEncryptionHandler create a Standard security handler
// and install it on the returned encrypt dict.
// The field V and P of the encrypt dict must be setup previously.
// `userPassword` and `ownerPassword` are used to generate the encryption keys
// and will be needed to decrypt the document.
func (d Document) UseStandardEncryptionHandler(enc Encrypt, ownerPassword, userPassword string, encryptMetadata bool) Encrypt {
	enc.Filter = "Standard"
	enc.SubFilter = ""

	var revision uint8
	if enc.V < 2 && !enc.P.isRevision3() {
		revision = 2
	} else if enc.V == 2 || enc.V == 3 || enc.P.isRevision3() {
		revision = 3
	} else if enc.V == EaRC4Custom {
		revision = 4
	} else if enc.V == EaAES {
		revision = 5
	}

	s := enc.NewRC4SecurityHandler(d.Trailer.ID[0], revision, !encryptMetadata)

	var out EncryptionStandard
	out.R = s.revision
	out.DontEncryptMetadata = s.dontEncryptMetadata

	out.O = s.generateOwnerHash(userPassword, ownerPassword)
	out.encryptionKey = s.generateEncryptionKey(userPassword, out.O)
	out.U = s.generateUserHash(out.encryptionKey)

	enc.EncryptionHandler = out

	return enc
}

func (e EncryptionStandard) encryptionAddFields() string {
	hashLength := 32
	if e.R == 5 {
		hashLength = 48
	}
	out := fmt.Sprintf("/R %d /O %s /U %s /EncryptMetadata %v",
		e.R, EscapeByteString(e.O[:hashLength]),
		EscapeByteString(e.U[:hashLength]), !e.DontEncryptMetadata)
	if e.R == 5 {
		out += fmt.Sprintf("/UE %s /OE %s", EspaceHexString(e.UE[:]), EspaceHexString(e.OE[:]))
	}
	return out
}

func (e EncryptionStandard) Clone() EncryptionHandler {
	out := e
	out.encryptionKey = append([]byte(nil), e.encryptionKey...)
	return out
}

// crypt encrypt in-place the given `data` using its object number,
// with the RC4 algorithm.
func (p EncryptionStandard) crypt(n Reference, data []byte) ([]byte, error) {
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

func padPassword(password string) (out [32]byte) {
	copy(out[:], append([]byte(password), padding[:]...)[0:32])
	return out
}

// key length in bytes, between
func (s *RC4SecurityHandler) keyLength() int {
	if s.revision >= 3 && s.specifiedKeyLength != 0 {
		return s.specifiedKeyLength
	}
	return 5
}

// xor19Times performs the additional step required by security handlers with revision >= 3.
//
// From the SPEC :
// Do the following 19 times: Take the output from the previous
// invocation of the RC4 function and pass it as input to a new invocation of the function; use an encryption
// key generated by taking each byte of the encryption key obtained in step (d) and performing an XOR
// (exclusive or) operation between that byte and the single-byte value of the iteration counter (from 1 to 19).
//
// `content` is updated in place
func xor19Times(content []byte, baseEncKey []byte) {
	newKey := make([]byte, len(baseEncKey)) // copy to preserve baseEncKey
	for i := byte(1); i <= 19; i++ {
		for j, b := range baseEncKey { // update the encKey
			newKey[j] = b ^ i
		}
		c, _ := rc4.NewCipher(newKey)
		c.XORKeyStream(content, content)
	}
}

// ------------------------------------------------------------------------------------

// crypt is not supported for the PublicKey security handler
// Thus, this function return the plain data.
func (e EncryptionPublicKey) crypt(n Reference, data []byte) ([]byte, error) {
	return data, nil
}
