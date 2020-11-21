package reader

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/benoitkugler/pdf/model"
)

func TestStructureTree(t *testing.T) {
	nbStruct := 0
	nbIds := 0
	var walkStruct func(*model.StructureElement)
	walkStruct = func(m *model.StructureElement) {
		nbStruct++
		for _, k := range m.K {
			if s, ok := k.(*model.StructureElement); ok {
				walkStruct(s)
			}
		}
		for _, att := range m.A {
			for name := range att.Attributes {
				if name == "ID" {
					nbIds++
				}
			}
		}
	}
	for _, s := range pdfSpec.Catalog.StructTreeRoot.K {
		walkStruct(s)
	}
	fmt.Println("Total number of structures elements", nbStruct)

	fmt.Println("ID fields in custom attributes", nbIds)

	d1 := pdfSpec.Catalog.StructTreeRoot.IDTree.LookupTable()
	fmt.Println("Original id tree total size", len(d1))

	ti := time.Now()
	pdfSpec.Catalog.StructTreeRoot.BuildIDTree()
	fmt.Println("	Building IDTree in", time.Since(ti))

	d2 := pdfSpec.Catalog.StructTreeRoot.IDTree.LookupTable()
	fmt.Println("Automatic id tree total size", len(d2))

	if !reflect.DeepEqual(d1, d2) {
		t.Errorf("expected %v, got %v", d1, d2)
	}

	d3 := pdfSpec.Catalog.StructTreeRoot.ParentTree.LookupTable()
	fmt.Println("Original parent tree total size", len(d3))

	ti = time.Now()
	pdfSpec.Catalog.StructTreeRoot.BuildParentTree()
	fmt.Println("	Building ParentTree in", time.Since(ti))

	d4 := pdfSpec.Catalog.StructTreeRoot.ParentTree.LookupTable()
	fmt.Println("Automatic parent tree total size", len(d4))

	// the StructParent 4171 is broken in the PDF spec
	delete(d3, 4717)

	// we need to compare not taking list order into account
	if len(d3) != len(d4) {
		t.Error("expected same maps")
	}
	for n := range d3 {
		n1, n2 := d3[n], d4[n]
		if n1.Num != n2.Num || n1.Parent != n2.Parent {
			t.Errorf("expected %v got %v", n1, n2)
		}
		if len(n1.Parents) != len(n2.Parents) {
			t.Errorf("expected %v got %v", n1, n2)
		}
		m1 := map[*model.StructureElement]bool{}
		m2 := map[*model.StructureElement]bool{}
		for i := range n1.Parents {
			m1[n1.Parents[i]] = true
			m2[n2.Parents[i]] = true
		}
		if !reflect.DeepEqual(m1, m2) {
			t.Errorf("expected %v got %v", n1, n2)
		}
	}
}
