package model_test

import (
	"bytes"
	"crypto/rc4"
	"os"
	"strings"
	"testing"

	mo "github.com/benoitkugler/pdf/model"
	"github.com/benoitkugler/pdf/reader/file"
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
	var doc mo.Document
	doc.Catalog.Pages.Kids = []mo.PageNode{&mo.PageObject{Contents: []mo.ContentStream{
		{Stream: mo.Stream{Content: []byte(strings.Repeat("dlmskd", 10))}},
	}}}
	up, op := "dlà&#mks", "elmzk89.ek"
	for _, v := range [...]mo.EncryptionAlgorithm{mo.EaRC440, mo.EaRC4Ext} {
		for _, p := range [...]mo.UserPermissions{
			mo.PermissionPrint,
			mo.PermissionModify,
			mo.PermissionCopy,
			mo.PermissionAdd,
			mo.PermissionFill,
			mo.PermissionExtract,
			mo.PermissionAssemble,
			mo.PermissionPrintDigital,
		} {
			enc := mo.Encrypt{V: v, P: p}
			enc = doc.UseStandardEncryptionHandler(enc, op, up, true)
			f, err := os.Create("test/rc4.pdf")
			if err != nil {
				t.Fatal(err)
			}
			err = doc.Write(f, &enc)
			if err != nil {
				t.Error(err)
			}
			f.Close()

			_, err = file.ReadFile("test/rc4.pdf", &file.Configuration{Password: up})
			if err != nil {
				t.Error(err)
			}
			_, err = file.ReadFile("test/rc4.pdf", &file.Configuration{Password: op})
			if err != nil {
				t.Error(err)
			}
			_, err = file.ReadFile("test/rc4.pdf", &file.Configuration{Password: op + "4"})
			if err == nil {
				t.Errorf("expected error")
			}
			_, err = file.ReadFile("test/rc4.pdf", &file.Configuration{Password: up + "4"})
			if err == nil {
				t.Errorf("expected error")
			}
		}
	}
}
