package cmaps

import (
	"fmt"
	"os"
	"testing"

	"github.com/benoitkugler/pdf/model"
)

// cmap1Data represents a basic CMap.
const cmap1Data = `
	/CIDInit/ProcSet findresource begin
	12 dict begin
	begincmap
	/CIDSystemInfo
	<< /Registry (Adobe)
	/Ordering (UCS)
	/Supplement 0
	>> def
	/CMapName/Adobe-Identity-UCS def
	/CMapType 2 def
	1 begincodespacerange
	<0000> <FFFF>
	endcodespacerange
	8 beginbfchar
	<0003> <0020>
	<0007> <0024>
	<0033> <0050>
	<0035> <0052>
	<0037> <0054>
	<005A> <0077>
	<005C> <0079>
	<005F> <007C>
	endbfchar
	7 beginbfrange
	<000F> <0017> <002C>
	<001B> <001D> <0038>
	<0025> <0026> <0042>
	<002F> <0031> <004C>
	<0044> <004C> <0061>
	<004F> <0053> <006C>
	<0055> <0057> <0072>
	endbfrange
	endcmap
	CMapName currentdict/CMap defineresource pop
	end
	end
`

// TestParser tests basic loading of a simple CMap.
func TestParser1(t *testing.T) {
	cmap := newparser([]byte(cmap1Data))
	err := cmap.parse()
	if err != nil {
		t.Error("Failed: ", err)
		return
	}
	cmap.computeInverseMappings()

	if cmap.cids.Name != "Adobe-Identity-UCS" {
		t.Errorf("CMap name incorrect (%s)", cmap.cids.Name)
		return
	}

	if cmap.cids.Type != 2 {
		t.Errorf("CMap type incorrect")
		return
	}

	if len(cmap.cids.Codespaces) != 1 {
		t.Errorf("len codespace != 1 (%d)", len(cmap.cids.Codespaces))
		return
	}

	if cmap.cids.Codespaces[0].Low != 0 {
		t.Errorf("code space low range != 0 (%d)", cmap.cids.Codespaces[0].Low)
		return
	}

	if cmap.cids.Codespaces[0].High != 0xFFFF {
		t.Errorf("code space high range != 0xffff (%d)", cmap.cids.Codespaces[0].High)
		return
	}

	expectedMappings := map[model.CID]rune{
		0x0003:     0x0020,
		0x005F:     0x007C,
		0x000F:     0x002C,
		0x000F + 5: 0x002C + 5,
		0x001B:     0x0038,
		0x001B + 2: 0x0038 + 2,
		0x002F:     0x004C,
		0x0044:     0x0061,
		0x004F:     0x006C,
		0x0055:     0x0072,
	}

	for k, expected := range expectedMappings {
		v, ok := cmap.unicode.ProperLookupTable()[k]
		if !ok || len(v) != 1 || v[0] != expected {
			t.Errorf("incorrect mapping, expecting 0x%X ➞ 0x%X (%#v)", k, expected, v)
			return
		}
	}

	_, ok := cmap.unicode.ProperLookupTable()[0x99]
	if ok { //!= "notdef" {
		t.Errorf("Unmapped code, expected to map to undefined")
		return
	}

	// charcodes := []byte{0x00, 0x03, 0x00, 0x0F}
	// s, _ := cmap.CharcodeBytesToUnicode(charcodes)
	// if s != " ," {
	// 	t.Error("Incorrect charcode bytes ➞ string mapping")
	// 	return
	// }
}

const cmap2Data = `
	/CIDInit/ProcSet findresource begin
	12 dict begin
	begincmap
	/CIDSystemInfo
	<< /Registry (Adobe)
	/Ordering (UCS)
	/Supplement 0
	>> def
	/CMapName/Adobe-Identity-UCS def
	/CMapType 2 def
	1 begincodespacerange
	<0000> <FFFF>
	endcodespacerange
	7 beginbfrange
	<0080> <00FF> <002C>
	<802F> <902F> <0038>
	endbfrange
	endcmap
	CMapName currentdict/CMap defineresource pop
	end
	end
`

// TestParser2 tests a bug that came up when 2-byte character codes had the higher byte set to 0,
// e.g. 0x0080, and the character map was not taking the number of bytes of the input codemap into account.
func TestParser2(t *testing.T) {
	cmap := newparser([]byte(cmap2Data))
	err := cmap.parse()
	if err != nil {
		t.Error("Failed: ", err)
		return
	}
	cmap.computeInverseMappings()

	if cmap.cids.Name != "Adobe-Identity-UCS" {
		t.Errorf("CMap name incorrect (%s)", cmap.cids.Name)
		return
	}

	if cmap.cids.Type != 2 {
		t.Errorf("CMap type incorrect")
		return
	}

	if len(cmap.cids.Codespaces) != 1 {
		t.Errorf("len codespace != 1 (%d)", len(cmap.cids.Codespaces))
		return
	}

	if cmap.cids.Codespaces[0].Low != 0 {
		t.Errorf("code space low range != 0 (%d)", cmap.cids.Codespaces[0].Low)
		return
	}

	if cmap.cids.Codespaces[0].High != 0xFFFF {
		t.Errorf("code space high range != 0xffff (%d)", cmap.cids.Codespaces[0].High)
		return
	}

	expectedMappings := map[model.CID]rune{
		0x0080: 0x002C,
		0x802F: 0x0038,
	}

	for k, expected := range expectedMappings {
		v, ok := cmap.unicode.ProperLookupTable()[k]
		if !ok || len(v) != 1 || v[0] != expected {
			t.Errorf("incorrect mapping, expecting 0x%X ➞ 0x%X (got 0x%X)", k, expected, v)
			return
		}
	}

	// // Check byte sequence mappings.
	// expectedSequenceMappings := []struct {
	// 	bytes    []byte
	// 	expected string
	// }{
	// 	{[]byte{0x80, 0x2F, 0x00, 0x80}, string([]rune{0x0038, 0x002C})},
	// }

	// for _, exp := range expectedSequenceMappings {
	// 	str, _ := cmap.CharcodeBytesToUnicode(exp.bytes)
	// 	if str != exp.expected {
	// 		t.Errorf("Incorrect byte sequence mapping % X ➞ % X (got % X)",
	// 			exp.bytes, []rune(exp.expected), []rune(str))
	// 		return
	// 	}
	// }
}

// cmap3Data is a CMap with a mixture of 1 and 2 byte codespaces.
const cmap3Data = `
	/CIDInit/ProcSet findresource begin
	12 dict begin begincmap
	/CIDSystemInfo
	3 dict dup begin
	/Registry (Adobe) def
	/Supplement 2 def
	end def

	/CMapName/test-1 def
	/CMapType 1 def

	4 begincodespacerange
	<00> <80>
	<8100> <9fff>
	<a0> <d0>
	<d140> <fbfc>
	endcodespacerange
	endcmap
`

// TestParser3 test case of a CMap with mixed number of 1 and 2 bytes in the codespace range.
func TestParser3(t *testing.T) {
	cmap, err := ParseCIDCMap([]byte(cmap3Data))
	if err != nil {
		t.Fatal("Failed: ", err)
	}

	if cmap.Name != "test-1" {
		t.Fatalf("CMap name incorrect (%s)", cmap.Name)
	}

	if cmap.Type != 1 {
		t.Fatalf("CMap type incorrect")
	}

	// Check codespaces.
	expectedCodespaces := []Codespace{
		{NumBytes: 1, Low: 0x00, High: 0x80},
		{NumBytes: 1, Low: 0xa0, High: 0xd0},
		{NumBytes: 2, Low: 0x8100, High: 0x9fff},
		{NumBytes: 2, Low: 0xd140, High: 0xfbfc},
	}

	if len(cmap.Codespaces) != len(expectedCodespaces) {
		t.Fatalf("len codespace != %d (%d)", len(expectedCodespaces), len(cmap.Codespaces))
	}

	for i, cs := range cmap.Codespaces {
		exp := expectedCodespaces[i]
		if cs.NumBytes != exp.NumBytes {
			t.Fatalf("code space number of bytes != %d (%d) %x", exp.NumBytes, cs.NumBytes, exp)
		}

		if cs.Low != exp.Low {
			t.Fatalf("code space low range != %d (%d) %x", exp.Low, cs.Low, exp)
		}

		if cs.High != exp.High {
			t.Fatalf("code space high range != 0x%X (0x%X) %x", exp.High, cs.High, exp)
		}
	}

	// // Check byte sequence mappings.
	// expectedSequenceMappings := []struct {
	// 	bytes    []byte
	// 	expected string
	// }{

	// 	{[]byte{0x80, 0x81, 0x00, 0xa1, 0xd1, 0x80, 0x00},
	// 		string([]rune{
	// 			0x90,
	// 			0x1000,
	// 			0x91,
	// 			0xa000 + 0x40,
	// 			0x10})},
	// }

	// for _, exp := range expectedSequenceMappings {
	// 	str, _ := cmap.CharcodeBytesToUnicode(exp.bytes)
	// 	if str != exp.expected {
	// 		t.Errorf("Incorrect byte sequence mapping: % 02X ➞ % 02X (got % 02X)",
	// 			exp.bytes, []rune(exp.expected), []rune(str))
	// 		return
	// 	}
	// }
}

// cmapData4 is a CMap with some utf16 encoded unicode strings that contain surrogates.
const cmap4Data = `
   /CIDInit/ProcSet findresource begin
    11 dict begin
    begincmap
   /CIDSystemInfo
    <</Registry (Adobe)
   /Ordering (UCS)
   /Supplement 0
    >> def
   /CMapName/Adobe-Identity-UCS def
   /CMapType 2 def
    1 begincodespacerange
    <0000> <FFFF>
    endcodespacerange
    15 beginbfchar
    <01E1> <002C>
    <0201> <007C>
    <059C> <21D2>
    <05CA> <2200>
    <05CC> <2203>
    <05D0> <2208>
    <0652> <2295>
    <073F> <D835DC50>
    <0749> <D835DC5A>
    <0889> <D835DC84>
    <0893> <D835DC8E>
    <08DD> <D835DC9E>
    <08E5> <D835DCA6>
    <08E7> <2133>
    <0D52> <2265>
    endbfchar
    1 beginbfrange
    <0E36> <0E37> <27F5>
    endbfrange
    endcmap
`

// TestParser4 checks that ut16 encoded unicode strings are interpreted correctly.
func TestParser4(t *testing.T) {
	cmap := newparser([]byte(cmap4Data))
	err := cmap.parse()
	if err != nil {
		t.Error("Failed to load CMap: ", err)
		return
	}
	cmap.computeInverseMappings()

	if cmap.cids.Name != "Adobe-Identity-UCS" {
		t.Errorf("CMap name incorrect (%s)", cmap.cids.Name)
		return
	}

	if cmap.cids.Type != 2 {
		t.Errorf("CMap type incorrect")
		return
	}

	if len(cmap.cids.Codespaces) != 1 {
		t.Errorf("len codespace != 1 (%d)", len(cmap.cids.Codespaces))
		return
	}

	if cmap.cids.Codespaces[0].Low != 0 {
		t.Errorf("code space low range != 0 (%d)", cmap.cids.Codespaces[0].Low)
		return
	}

	if cmap.cids.Codespaces[0].High != 0xFFFF {
		t.Errorf("code space high range != 0xffff (%d)", cmap.cids.Codespaces[0].High)
		return
	}

	expectedMappings := map[model.CID]rune{
		0x0889: '\U0001d484', // `𝒄`
		0x0893: '\U0001d48e', // `𝒎`
		0x08DD: '\U0001d49e', // `𝒞`
		0x08E5: '\U0001d4a6', // `𝒦
	}

	for k, expected := range expectedMappings {
		v, ok := cmap.unicode.ProperLookupTable()[k]
		if !ok || len(v) != 1 || v[0] != expected {
			t.Errorf("incorrect mapping, expecting 0x%04X ➞ %+q (got %+q)", k, expected, v)
			return
		}
	}

	// // Check byte sequence mappings.
	// expectedSequenceMappings := []struct {
	// 	bytes    []byte
	// 	expected string
	// }{
	// 	{[]byte{0x07, 0x3F, 0x07, 0x49}, "\U0001d450\U0001d45a"}, // `𝑐𝑚`
	// 	{[]byte{0x08, 0x89, 0x08, 0x93}, "\U0001d484\U0001d48e"}, // `𝒄𝒎`
	// 	{[]byte{0x08, 0xDD, 0x08, 0xE5}, "\U0001d49e\U0001d4a6"}, // `𝒞𝒦`
	// 	{[]byte{0x08, 0xE7, 0x0D, 0x52}, "\u2133\u2265"},         // `ℳ≥`
	// }

	// for _, exp := range expectedSequenceMappings {
	// 	str, _ := cmap.CharcodeBytesToUnicode(exp.bytes)
	// 	if str != exp.expected {
	// 		t.Errorf("Incorrect byte sequence mapping % 02X ➞ %+q (got %+q)",
	// 			exp.bytes, exp.expected, str)
	// 		return
	// 	}
	// }
}

func TestFullCIDCMap(t *testing.T) {
	names := [...]model.ObjName{"Adobe-CNS1-3", "KSCms-UHC-H", "Ext-RKSJ-V"}
	nbCidRanges := [...]int{74, 675, 1}
	for i, file := range [...]string{
		"test/Adobe-CNS1-3.cmap",
		"test/KSCms-UHC-H.cmap",
		"test/usecmap.cmap",
	} {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		cmap, err := ParseCIDCMap(b)
		if err != nil {
			t.Fatal(err)
		}
		if cmap.Name != names[i] {
			t.Errorf("expected %s got %s", names[i], cmap.Name)
		}
		if L := len(cmap.CIDs); L != nbCidRanges[i] {
			t.Errorf("expected %d ranges, got %d", nbCidRanges[i], L)
		}
	}
}

func TestFullToUnicodeCMap(t *testing.T) {
	for _, file := range [...]string{
		"../standardcmaps/generate/Adobe-CNS1-UCS2.txt",
		"../standardcmaps/generate/Adobe-GB1-UCS2.txt",
		"../standardcmaps/generate/Adobe-Japan1-UCS2.txt",
		"../standardcmaps/generate/Adobe-Korea1-UCS2.txt",
		"../standardcmaps/generate/Adobe-KR-UCS2.txt",
	} {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		cmap, err := ParseUnicodeCMap(b)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(len(cmap.ProperLookupTable()))
	}
}
