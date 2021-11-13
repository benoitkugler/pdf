package file

import (
	"os"
	"strings"
	"testing"

	"github.com/benoitkugler/pdf/model"
)

func TestReadProtectedRC4(t *testing.T) {
	file := "../test/ProtectedRC4.pdf"
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	_, err = Read(f, nil)
	if _, ok := err.(IncorrectPasswordErr); !ok {
		t.Fatalf("expected error for invalid password, got %v", err)
	}

	// we check that both passwords are valid

	userPassword, ownerPassword := "78eoln_-(_àç_-')", "aaaa"
	out, err := Read(f, &Configuration{Password: userPassword})
	if err != nil {
		t.Fatal(err)
	}

	out, err = Read(f, &Configuration{Password: ownerPassword})
	if err != nil {
		t.Fatal(err)
	}

	st := out.ResolveObject(model.ObjIndirectRef{ObjectNumber: 4}).(model.ObjStream)
	if !strings.Contains(string(st.Content), "My secret content !") {
		t.Fatalf("unexepected content %s", st.Content)
	}
}

func TestReadProtectedAES(t *testing.T) {
	file := "../test/ProtectedAES.pdf"
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	_, err = Read(f, nil)
	if _, ok := err.(IncorrectPasswordErr); !ok {
		t.Fatalf("expected error for invalid password, got %v", err)
	}

	_, err = Read(f, &Configuration{Password: "aaaa"})
	if err != nil {
		t.Fatal(err)
	}
}
