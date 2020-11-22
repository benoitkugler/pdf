package model

import (
	"reflect"
	"strings"
	"testing"
)

func TestChunks(t *testing.T) {
	a := make([]int, 20)
	for i := range a {
		a[i] = i
	}
	chunks := splitInts(a, 3)
	if len(chunks) != 7 {
		t.Errorf("expected 7 chunks, got %d", len(chunks))
	}

	chunks = splitInts(a, 4)
	if len(chunks) != 5 {
		t.Errorf("expected 5 chunks, got %d", len(chunks))
	}

	b := make([]string, 18)
	for i := range b {
		b[i] = strings.Repeat("u", i)
	}
	chunks2 := splitStrings(b, 3)
	if len(chunks2) != 6 {
		t.Errorf("expected 6 chunks, got %d", len(chunks))
	}

	chunks2 = splitStrings(b, 4)
	if len(chunks2) != 5 {
		t.Errorf("expected 5 chunks, got %d", len(chunks))
	}
}

func TestIDTree(t *testing.T) {
	m := make(map[string]*StructureElement)

	for i := range [1000]int{} {
		m[strings.Repeat("s", i)] = new(StructureElement)
	}

	tree := NewIDTree(m)
	m2 := tree.LookupTable()
	if !reflect.DeepEqual(m, m2) {
		t.Errorf("expected %v, got %v", m, m2)
	}
}

func TestParentTree(t *testing.T) {
	m := make(map[int]NumToParent)

	for i := range [1000]int{} {
		m[i] = NumToParent{Num: i, Parent: new(StructureElement)}
	}

	tree := NewParentTree(m)
	m2 := tree.LookupTable()
	if !reflect.DeepEqual(m, m2) {
		t.Errorf("expected %v, got %v", m, m2)
	}
}
