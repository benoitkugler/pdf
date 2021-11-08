package file

import (
	"strings"
	"testing"
)

func TestReadBlindly(t *testing.T) {
	rs := strings.NewReader(strings.Repeat("1234", 1000) + "as4 endstream")
	ctx, _ := newContext(rs, nil)
	out, err := ctx.readStreamBlindly(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 4*1000+4 {
		t.Fatalf("unexpected content %s", out)
	}

	rs = strings.NewReader("as4 endstream")
	ctx, _ = newContext(rs, nil)
	out, err = ctx.readStreamBlindly(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 4 {
		t.Fatalf("unexpected content %s", out)
	}

	rs = strings.NewReader(strings.Repeat("1234", 1000) + "as4 ")
	ctx, _ = newContext(rs, nil)
	_, err = ctx.readStreamBlindly(0)
	if err == nil {
		t.Fatal("expected error on EOF")
	}
}

func TestReadMaxLength(t *testing.T) {
	rs := strings.NewReader(strings.Repeat("1234", 1000) + "as4 endstream")
	ctx, _ := newContext(rs, nil)
	out, err := ctx.readStreamMaxLength(0, 4*1000+4)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 4*1000+4 {
		t.Fatalf("unexpected content %s", out)
	}

	rs = strings.NewReader("as4 endstream")
	ctx, _ = newContext(rs, nil)
	out, err = ctx.readStreamMaxLength(0, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 4 {
		t.Fatalf("unexpected content %s", out)
	}

	rs = strings.NewReader("as4 endstream")
	ctx, _ = newContext(rs, nil)
	out, err = ctx.readStreamMaxLength(0, 15)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 4 {
		t.Fatalf("unexpected content %s", out)
	}

	rs = strings.NewReader("as4 ")
	ctx, _ = newContext(rs, nil)
	_, err = ctx.readStreamMaxLength(0, 45)
	if err == nil {
		t.Fatal("expected error on EOF")
	}
}
