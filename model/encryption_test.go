package model

import (
	"crypto/rc4"
	"fmt"
	"testing"
)

func TestOverlap(t *testing.T) {
	rc, _ := rc4.NewCipher([]byte("medlùl"))
	in := []byte("ldsqdlqsùdl")
	out := make([]byte, len(in))
	rc.XORKeyStream(out, in)
	fmt.Println(out)

	rc, _ = rc4.NewCipher([]byte("medlùl"))
	fmt.Println(in)
	rc.XORKeyStream(in, in)
	fmt.Println(in)
}

func TestRC4(t *testing.T) {
	tr := Trailer{Encrypt: Encrypt{V: 1, P: PermissionCopy | PermissionFill}}
	tr.SetStandardEncryptionHandler("mdsl", "lmkemke", true)
	d := []byte("ezelùmelùsùsdmsdùsùù	amùsùmzezmezùeùlùm")
	tr.Encrypt.EncryptionHandler.Crypt(45, d)
	fmt.Println(string(d))
}
