package model

import (
	"bytes"
	"crypto/rc4"
	"os"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func TestOverlap(t *testing.T) {
	rc, _ := rc4.NewCipher([]byte("medlùl"))
	in := []byte("ldsqdlqsùdl")
	out := make([]byte, len(in))
	rc.XORKeyStream(out, in)

	rc, _ = rc4.NewCipher([]byte("medlùl"))
	rc.XORKeyStream(in, in)
	if !bytes.Equal(out, in) {
		t.Errorf("expected same output, got %v and %v", out, in)
	}
}

func TestRC4Basic(t *testing.T) {
	var doc Document
	doc.Catalog.Pages.Kids = []PageNode{&PageObject{Contents: []ContentStream{
		{Stream: Stream{Content: []byte(strings.Repeat("dlmskd", 10))}},
	}}}
	up, op := "dlà&#mks", "elmzk89.ek"
	for _, v := range [...]EncryptionAlgorithm{Key40, KeyExt} {
		for _, p := range [...]UserPermissions{
			PermissionPrint,
			PermissionModify,
			PermissionCopy,
			PermissionAdd,
			PermissionFill,
			PermissionExtract,
			PermissionAssemble,
			PermissionPrintDigital,
		} {
			enc := Encrypt{V: v, P: p}
			enc = doc.UseStandardEncryptionHandler(enc, up, op, true)
			f, err := os.Create("test/rc4.pdf")
			if err != nil {
				t.Fatal(err)
			}
			err = doc.Write(f, &enc)
			if err != nil {
				t.Error(err)
			}
			f.Close()

			_, err = pdfcpu.ReadFile("test/rc4.pdf", &pdfcpu.Configuration{Reader15: true, UserPW: up})
			if err != nil {
				t.Error(err)
			}
			_, err = pdfcpu.ReadFile("test/rc4.pdf", &pdfcpu.Configuration{Reader15: true, OwnerPW: op})
			if err != nil {
				t.Error(err)
			}
			_, err = pdfcpu.ReadFile("test/rc4.pdf", &pdfcpu.Configuration{Reader15: true, OwnerPW: op + "4"})
			if err == nil {
				t.Errorf("expected error")
			}
			_, err = pdfcpu.ReadFile("test/rc4.pdf", &pdfcpu.Configuration{Reader15: true, UserPW: up + "4"})
			if err == nil {
				t.Errorf("expected error")
			}
		}
	}
}
