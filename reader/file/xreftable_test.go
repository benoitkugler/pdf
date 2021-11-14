package file

import (
	"os"
	"testing"
)

func TestXrefStream(t *testing.T) {
	src, err := os.Open("../test/corpus/UTF-32.pdf")
	if err != nil {
		t.Fatal(err)
	}

	ctx, err := processFile(src, nil)
	if err != nil {
		t.Fatal(err)
	}

	for obj, entry := range ctx.xrefTable.objects {
		if entry.free {
			continue
		}

		expectedOffset := expected[obj.ObjectNumber]
		if entry.offset != expectedOffset {
			t.Fatalf("for object %d, expected %d, got %d", obj.ObjectNumber, expectedOffset, entry.offset)
		}
	}
}
