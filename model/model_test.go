package model

import (
	"bytes"
	"reflect"
	"testing"
)

func TestEmptyDocument(t *testing.T) {
	var d Document
	var b bytes.Buffer
	err := d.Write(&b)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Bytes()) != 297 {
		t.Fatalf("expected 297 bytes for an empty Document, got %d", len(b.Bytes()))
	}
}

func TestCloneTrailer(t *testing.T) {
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

	tr := Trailer{Encrypt: e}
	tr2 := tr.Clone()
	if !reflect.DeepEqual(tr, tr2) {
		t.Errorf("expected %v, got %v", tr, tr2)
	}
}

func TestCloneNames(t *testing.T) {
	e := NameDictionary{
		EmbeddedFiles: EmbeddedFileTree{
			{Name: "ùmlld", FileSpec: &FileSpec{}},
		},
		Dests: &DestTree{
			Kids: []DestTree{
				{Names: []NameToDest{
					{Name: "mùdlsld", Destination: nil},
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
