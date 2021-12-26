package model

import (
	"bytes"
	"os"
	"reflect"
	"testing"
)

func TestEmptyDocument(t *testing.T) {
	var d Document
	var b bytes.Buffer
	err := d.Write(&b, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Bytes()) != 285 {
		t.Fatalf("expected 285 bytes for an empty Document, got %d", len(b.Bytes()))
	}
}

func TestCloneEncrypt(t *testing.T) {
	e := Encrypt{
		CF: map[Name]CrypFilter{
			"lmds": {},
			"sdsd": {
				Recipients: []string{"ds5"},
			},
		},
	}
	e2 := e.Clone()
	if !reflect.DeepEqual(e, e2) {
		t.Errorf("expected %v, got %v", e, e2)
	}
}

func TestCloneNames(t *testing.T) {
	e := NameDictionary{
		EmbeddedFiles: EmbeddedFileTree{
			{Name: "ùmlld", FileSpec: &FileSpec{}},
		},
		Dests: DestTree{
			Kids: []DestTree{
				{
					Names: []NameToDest{
						{Name: "mùdlsld", Destination: DestinationExplicitIntern{}},
					},
				},
			},
			Names: []NameToDest{
				{},
			},
		},
	}
	cache := newCloneCache()
	e2 := e.clone(cache)
	if !reflect.DeepEqual(e, e2) {
		t.Errorf("expected %v, got %v", e, e2)
	}
}

func TestCloneDocument(t *testing.T) {
	var doc Document

	if clone := doc.Clone(); !reflect.DeepEqual(doc, clone) {
		t.Errorf("expected %v, got %v", doc, clone)
	}
}

func TestOpenAction(t *testing.T) {
	var d Document
	p3 := &PageObject{}
	d.Catalog.Pages.Kids = []PageNode{
		&PageObject{},
		&PageObject{},
		p3,
		&PageTree{
			Kids: []PageNode{&PageObject{}},
		},
	}
	d.Catalog.OpenAction = Action{ActionType: ActionGoTo{D: DestinationExplicitIntern{
		Page: p3, Location: DestinationLocationFit("Fit"),
	}}}
	f, err := os.Create("test/open_action.pdf")
	if err != nil {
		t.Fatal(err)
	}
	err = d.Write(f, nil)
	if err != nil {
		t.Fatal(err)
	}
}
