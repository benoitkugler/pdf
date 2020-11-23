package model

import (
	"fmt"
	"reflect"
	"testing"

	"golang.org/x/text/encoding/charmap"
)

func TestEncodingString(t *testing.T) {
	diffs := Differences{
		1:  "dsd",
		2:  "mldsks",
		3:  "mdsùldùs",
		10: "ee",
		11: "sd",
		12: "ee",
		7:  "88",
	}
	exp := "[ 1/dsd/mldsks/mdsùldùs 7/88 10/ee/sd/ee]"
	if d := diffs.PDFString(); d != exp {
		t.Errorf("expected %s, got %v", exp, d)
	}
}

func TestCloneFont(t *testing.T) {
	fonts := []Font{
		FontType0{Encoding: CMapEncodingEmbedded{}},
		FontType0{Encoding: CMapEncodingPredefined("")},
		FontType1{
			Encoding: MacRomanEncoding,
		},
		FontTrueType{
			Widths: make([]int, 12),
		},
		FontType3{
			Encoding:  MacExpertEncoding,
			CharProcs: map[Name]ContentStream{"mkdmsk": {}},
		},
	}
	c := cloneCache{}
	for _, f := range fonts {
		f2 := f.clone(c)
		if !reflect.DeepEqual(f, f2) {
			t.Fatalf("expected deep equality, got %v and %v", f, f2)
		}
	}
}

var macExpertCharToRune = map[byte]rune{ // 165 entries
	0x20: ' ', 0x21: '\uf721', 0x22: '\uf6f8', 0x23: '\uf7a2', 0x24: '\uf724',
	0x25: '\uf6e4', 0x26: '\uf726', 0x27: '\uf7b4', 0x28: '⁽', 0x29: '⁾',
	0x2a: '‥', 0x2b: '․', 0x2c: ',', 0x2d: '-', 0x2e: '.',
	0x2f: '⁄', 0x30: '\uf730', 0x31: '\uf731', 0x32: '\uf732', 0x33: '\uf733',
	0x34: '\uf734', 0x35: '\uf735', 0x36: '\uf736', 0x37: '\uf737', 0x38: '\uf738',
	0x39: '\uf739', 0x3a: ':', 0x3b: ';', 0x3d: '\uf6de', 0x3f: '\uf73f',
	0x44: '\uf7f0', 0x47: '¼', 0x48: '½', 0x49: '¾', 0x4a: '⅛',
	0x4b: '⅜', 0x4c: '⅝', 0x4d: '⅞', 0x4e: '⅓', 0x4f: '⅔',
	0x56: 'ﬀ', 0x57: 'ﬁ', 0x58: 'ﬂ', 0x59: 'ﬃ', 0x5a: 'ﬄ',
	0x5b: '₍', 0x5d: '₎', 0x5e: '\uf6f6', 0x5f: '\uf6e5', 0x60: '\uf760',
	0x61: '\uf761', 0x62: '\uf762', 0x63: '\uf763', 0x64: '\uf764', 0x65: '\uf765',
	0x66: '\uf766', 0x67: '\uf767', 0x68: '\uf768', 0x69: '\uf769', 0x6a: '\uf76a',
	0x6b: '\uf76b', 0x6c: '\uf76c', 0x6d: '\uf76d', 0x6e: '\uf76e', 0x6f: '\uf76f',
	0x70: '\uf770', 0x71: '\uf771', 0x72: '\uf772', 0x73: '\uf773', 0x74: '\uf774',
	0x75: '\uf775', 0x76: '\uf776', 0x77: '\uf777', 0x78: '\uf778', 0x79: '\uf779',
	0x7a: '\uf77a', 0x7b: '₡', 0x7c: '\uf6dc', 0x7d: '\uf6dd', 0x7e: '\uf6fe',
	0x81: '\uf6e9', 0x82: '\uf6e0', 0x87: '\uf7e1', 0x88: '\uf7e0', 0x89: '\uf7e2',
	0x8a: '\uf7e4', 0x8b: '\uf7e3', 0x8c: '\uf7e5', 0x8d: '\uf7e7', 0x8e: '\uf7e9',
	0x8f: '\uf7e8', 0x90: '\uf7ea', 0x91: '\uf7eb', 0x92: '\uf7ed', 0x93: '\uf7ec',
	0x94: '\uf7ee', 0x95: '\uf7ef', 0x96: '\uf7f1', 0x97: '\uf7f3', 0x98: '\uf7f2',
	0x99: '\uf7f4', 0x9a: '\uf7f6', 0x9b: '\uf7f5', 0x9c: '\uf7fa', 0x9d: '\uf7f9',
	0x9e: '\uf7fb', 0x9f: '\uf7fc', 0xa1: '⁸', 0xa2: '₄', 0xa3: '₃',
	0xa4: '₆', 0xa5: '₈', 0xa6: '₇', 0xa7: '\uf6fd', 0xa9: '\uf6df',
	0xaa: '₂', 0xac: '\uf7a8', 0xae: '\uf6f5', 0xaf: '\uf6f0', 0xb0: '₅',
	0xb2: '\uf6e1', 0xb3: '\uf6e7', 0xb4: '\uf7fd', 0xb6: '\uf6e3', 0xb9: '\uf7fe',
	0xbb: '₉', 0xbc: '₀', 0xbd: '\uf6ff', 0xbe: '\uf7e6', 0xbf: '\uf7f8',
	0xc0: '\uf7bf', 0xc1: '₁', 0xc2: '\uf6f9', 0xc9: '\uf7b8', 0xcf: '\uf6fa',
	0xd0: '‒', 0xd1: '\uf6e6', 0xd6: '\uf7a1', 0xd8: '\uf7ff', 0xda: '¹',
	0xdb: '²', 0xdc: '³', 0xdd: '⁴', 0xde: '⁵', 0xdf: '⁶',
	0xe0: '⁷', 0xe1: '⁹', 0xe2: '⁰', 0xe4: '\uf6ec', 0xe5: '\uf6f1',
	0xe6: '\uf6f3', 0xe9: '\uf6ed', 0xea: '\uf6f2', 0xeb: '\uf6eb', 0xf1: '\uf6ee',
	0xf2: '\uf6fb', 0xf3: '\uf6f4', 0xf4: '\uf7af', 0xf5: '\uf6ea', 0xf6: 'ⁿ',
	0xf7: '\uf6ef', 0xf8: '\uf6e2', 0xf9: '\uf6e8', 0xfa: '\uf6f7', 0xfb: '\uf6fc',
}

func TestEncodeByte(t *testing.T) {
	// b := byte(0x91)
	r := '\u2018'
	fmt.Println(string(r), []byte(string(r)))
	r = 0x22000022
	fmt.Println(r, string(r), []byte(string(r)))
	var macEnc [256]rune
	for i := range macEnc {
		macEnc[i] = charmap.Macintosh.DecodeByte(byte(i))
	}
	fmt.Println(macEnc[0xc7], string(macEnc[0xc7]))

	var macExp [256]rune
	for i := range macExp {
		r1 := charmap.MacintoshCyrillic.DecodeByte(byte(i))
		r2 := macExpertCharToRune[byte(i)]
		fmt.Println(r1, r2)
	}
	fmt.Println(macExp[68], string(macExp[68]))
}
