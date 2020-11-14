package encodings

import "bytes"

// pdfDocEncoding defines the simple PdfDocEncoding.
var pdfDocEncoding = map[byte]rune{
	0x01: '\u0001', //     "controlSTX"
	0x02: '\u0002', //     "controlSOT"
	0x03: '\u0003', //     "controlETX"
	0x04: '\u0004', //     "controlEOT"
	0x05: '\u0005', //     "controlENQ"
	0x06: '\u0006', //     "controlACK"
	0x07: '\u0007', //     "controlBEL"
	0x08: '\u0008', //     "controlBS"
	0x09: '\u0009', //     "controlHT"
	0x0a: '\u000a', //     "controlLF"
	0x0b: '\u000b', //     "controlVT"
	0x0c: '\u000c', //     "controlFF"
	0x0d: '\u000d', //     "controlCR"
	0x0e: '\u000e', //     "controlSO"
	0x0f: '\u000f', //     "controlSI"
	0x10: '\u0010', //     "controlDLE"
	0x11: '\u0011', //     "controlDC1"
	0x12: '\u0012', //     "controlDC2"
	0x13: '\u0013', //     "controlDC3"
	0x14: '\u0014', //     "controlDC4"
	0x15: '\u0015', //     "controlNAK"
	0x16: '\u0017', //     "controlETB"
	0x17: '\u0017', //     "controlETB"
	0x18: '\u02d8', //  ˘  "breve"
	0x19: '\u02c7', //  ˇ  "caron"
	0x1a: '\u02c6', //  ˆ  "circumflex"
	0x1b: '\u02d9', //  ˙  "dotaccent"
	0x1c: '\u02dd', //  ˝  "hungarumlaut"
	0x1d: '\u02db', //  ˛  "ogonek"
	0x1e: '\u02da', //  ˚  "ring"
	0x1f: '\u02dc', //  ˜  "ilde"
	0x20: '\u0020', //     "space"
	0x21: '\u0021', //  !  "exclam"
	0x22: '\u0022', //  "  "quotedbl"
	0x23: '\u0023', //  #  "numbersign"
	0x24: '\u0024', //  $  "dollar"
	0x25: '\u0025', //  %  "percent"
	0x26: '\u0026', //  &  "ampersand"
	0x27: '\u0027', //  '  "quotesingle"
	0x28: '\u0028', //  (  "parenleft"
	0x29: '\u0029', //  )  "parenright"
	0x2a: '\u002a', //  *  "asterisk"
	0x2b: '\u002b', //  +  "plus"
	0x2c: '\u002c', //  ,  "comma"
	0x2d: '\u002d', //  -  "hyphen"
	0x2e: '\u002e', //  .  "period"
	0x2f: '\u002f', // /  "slash"
	0x30: '\u0030', //  0  "zero"
	0x31: '\u0031', //  1  "one"
	0x32: '\u0032', //  2  "two"
	0x33: '\u0033', //  3  "three"
	0x34: '\u0034', //  4  "four"
	0x35: '\u0035', //  5  "five"
	0x36: '\u0036', //  6  "six"
	0x37: '\u0037', //  7  "seven"
	0x38: '\u0038', //  8  "eight"
	0x39: '\u0039', //  9  "nine"
	0x3a: '\u003a', //  :  "colon"
	0x3b: '\u003b', //  ;  "semicolon"
	0x3c: '\u003c', //  <  "less"
	0x3d: '\u003d', //  =  "equal"
	0x3e: '\u003e', //  >  "greater"
	0x3f: '\u003f', //  ?  "question"
	0x40: '\u0040', //  @  "at"
	0x41: '\u0041', //  A  "A"
	0x42: '\u0042', //  B  "B"
	0x43: '\u0043', //  C  "C"
	0x44: '\u0044', //  D  "D"
	0x45: '\u0045', //  E  "E"
	0x46: '\u0046', //  F  "F"
	0x47: '\u0047', //  G  "G"
	0x48: '\u0048', //  H  "H"
	0x49: '\u0049', //  I  "I"
	0x4a: '\u004a', //  J  "J"
	0x4b: '\u004b', //  K  "K"
	0x4c: '\u004c', //  L  "L"
	0x4d: '\u004d', //  M  "M"
	0x4e: '\u004e', //  N  "N"
	0x4f: '\u004f', //  O  "O"
	0x50: '\u0050', //  P  "P"
	0x51: '\u0051', //  Q  "Q"
	0x52: '\u0052', //  R  "R"
	0x53: '\u0053', //  S  "S"
	0x54: '\u0054', //  T  "T"
	0x55: '\u0055', //  U  "U"
	0x56: '\u0056', //  V  "V"
	0x57: '\u0057', //  W  "W"
	0x58: '\u0058', //  X  "X"
	0x59: '\u0059', //  Y  "Y"
	0x5a: '\u005a', //  Z  "Z"
	0x5b: '\u005b', //  [  "bracketleft"
	0x5c: '\u005c', //  \  "backslash"
	0x5d: '\u005d', //  ]  "bracketright"
	0x5e: '\u005e', //  ^  "asciicircum"
	0x5f: '\u005f', //  _  "underscore"
	0x60: '\u0060', //  `  "grave"
	0x61: '\u0061', //  a  "a"
	0x62: '\u0062', //  b  "b"
	0x63: '\u0063', //  c  "c"
	0x64: '\u0064', //  d  "d"
	0x65: '\u0065', //  e  "e"
	0x66: '\u0066', //  f  "f"
	0x67: '\u0067', //  g  "g"
	0x68: '\u0068', //  h  "h"
	0x69: '\u0069', //  i  "i"
	0x6a: '\u006a', //  j  "j"
	0x6b: '\u006b', //  k  "k"
	0x6c: '\u006c', //  l  "l"
	0x6d: '\u006d', //  m  "m"
	0x6e: '\u006e', //  n  "n"
	0x6f: '\u006f', //  o  "o"
	0x70: '\u0070', //  p  "p"
	0x71: '\u0071', //  q  "q"
	0x72: '\u0072', //  r  "r"
	0x73: '\u0073', //  s  "s"
	0x74: '\u0074', //  t  "t"
	0x75: '\u0075', //  u  "u"
	0x76: '\u0076', //  v  "v"
	0x77: '\u0077', //  w  "w"
	0x78: '\u0078', //  x  "x"
	0x79: '\u0079', //  y  "y"
	0x7a: '\u007a', //  z  "z"
	0x7b: '\u007b', //  {  "braceleft"
	0x7c: '\u007c', //  |  "bar"
	0x7d: '\u007d', //  }  "braceright"
	0x7e: '\u007e', //  ~  "asciitilde"
	0x80: '\u2022', //  •  "bullet"
	0x81: '\u2020', //  †  "dagger"
	0x82: '\u2021', //  ‡  "daggerdbl"
	0x83: '\u2026', //  …  "ellipsis"
	0x84: '\u2014', //  —  "emdash"
	0x85: '\u2013', //  –  "endash"
	0x86: '\u0192', //  ƒ  "florin"
	0x87: '\u2044', //  ⁄  "fraction"
	0x88: '\u2039', //  ‹  "guilsinglleft"
	0x89: '\u203a', //  ›  "guilsinglright"
	0x8a: '\u2212', //  −  "minus"
	0x8b: '\u2030', //  ‰  "perthousand"
	0x8c: '\u201e', //  „  "quotedblbase"
	0x8d: '\u201c', //  “  "quotedblleft"
	0x8e: '\u201d', //  ”  "quotedblright"
	0x8f: '\u2018', //  ‘  "quoteleft"
	0x90: '\u2019', //  ’  "quoteright"
	0x91: '\u201a', //  ‚  "quotesinglbase"
	0x92: '\u2122', //  ™  "trademark"
	0x93: '\ufb01', //  ﬁ  "fi"
	0x94: '\ufb02', //  ﬂ  "fl"
	0x95: '\u0141', //  Ł  "Lslash"
	0x96: '\u0152', //  Œ  "OE"
	0x97: '\u0160', //  Š  "Scaron"
	0x98: '\u0178', //  Ÿ  "Ydieresis"
	0x99: '\u017d', //  Ž  "Zcaron"
	0x9a: '\u0131', //  ı  "dotlessi"
	0x9b: '\u0142', //  ł  "lslash"
	0x9c: '\u0153', //  œ  "oe"
	0x9d: '\u0161', //  š  "scaron"
	0x9e: '\u017e', //  ž  "zcaron"
	0xa0: '\u20ac', //  €  "Euro"
	0xa1: '\u00a1', //  ¡  "exclamdown"
	0xa2: '\u00a2', //  ¢  "cent"
	0xa3: '\u00a3', //  £  "sterling"
	0xa4: '\u00a4', //  ¤  "currency"
	0xa5: '\u00a5', //  ¥  "yen"
	0xa6: '\u00a6', //  ¦  "brokenbar"
	0xa7: '\u00a7', //  §  "section"
	0xa8: '\u00a8', //  ¨  "dieresis"
	0xa9: '\u00a9', //  ©  "copyright"
	0xaa: '\u00aa', //  ª  "ordfeminine"
	0xab: '\u00ab', //  «  "guillemotleft"
	0xac: '\u00ac', //  ¬  "logicalnot"
	0xae: '\u00ae', //  ®  "registered"
	0xaf: '\u00af', //  ¯  "macron"
	0xb0: '\u00b0', //  °  "degree"
	0xb1: '\u00b1', //  ±  "plusminus"
	0xb2: '\u00b2', //  ²  "twosuperior"
	0xb3: '\u00b3', //  ³  "threesuperior"
	0xb4: '\u00b4', //  ´  "acute"
	0xb5: '\u00b5', //  µ  "mu"
	0xb6: '\u00b6', //  ¶  "paragraph"
	0xb7: '\u00b7', //  ·  "middot"
	0xb8: '\u00b8', //  ¸  "cedilla"
	0xb9: '\u00b9', //  ¹  "onesuperior"
	0xba: '\u00ba', //  º  "ordmasculine"
	0xbb: '\u00bb', //  »  "guillemotright"
	0xbc: '\u00bc', //  ¼  "onequarter"
	0xbd: '\u00bd', //  ½  "onehalf"
	0xbe: '\u00be', //  ¾  "threequarters"
	0xbf: '\u00bf', //  ¿  "questiondown"
	0xc0: '\u00c0', //  À  "Agrave"
	0xc1: '\u00c1', //  Á  "Aacute"
	0xc2: '\u00c2', //  Â  "Acircumflex"
	0xc3: '\u00c3', //  Ã  "Atilde"
	0xc4: '\u00c4', //  Ä  "Adieresis"
	0xc5: '\u00c5', //  Å  "Aring"
	0xc6: '\u00c6', //  Æ  "AE"
	0xc7: '\u00c7', //  Ç  "Ccedilla"
	0xc8: '\u00c8', //  È  "Egrave"
	0xc9: '\u00c9', //  É  "Eacute"
	0xca: '\u00ca', //  Ê  "Ecircumflex"
	0xcb: '\u00cb', //  Ë  "Edieresis"
	0xcc: '\u00cc', //  Ì  "Igrave"
	0xcd: '\u00cd', //  Í  "Iacute"
	0xce: '\u00ce', //  Î  "Icircumflex"
	0xcf: '\u00cf', //  Ï  "Idieresis"
	0xd0: '\u00d0', //  Ð  "Eth"
	0xd1: '\u00d1', //  Ñ  "Ntilde"
	0xd2: '\u00d2', //  Ò  "Ograve"
	0xd3: '\u00d3', //  Ó  "Oacute"
	0xd4: '\u00d4', //  Ô  "Ocircumflex"
	0xd5: '\u00d5', //  Õ  "Otilde"
	0xd6: '\u00d6', //  Ö  "Odieresis"
	0xd7: '\u00d7', //  ×  "multiply"
	0xd8: '\u00d8', //  Ø  "Oslash"
	0xd9: '\u00d9', //  Ù  "Ugrave"
	0xda: '\u00da', //  Ú  "Uacute"
	0xdb: '\u00db', //  Û  "Ucircumflex"
	0xdc: '\u00dc', //  Ü  "Udieresis"
	0xdd: '\u00dd', //  Ý  "Yacute"
	0xde: '\u00de', //  Þ  "Thorn"
	0xdf: '\u00df', //  ß  "germandbls"
	0xe0: '\u00e0', //  à  "agrave"
	0xe1: '\u00e1', //  á  "aacute"
	0xe2: '\u00e2', //  â  "acircumflex"
	0xe3: '\u00e3', //  ã  "atilde"
	0xe4: '\u00e4', //  ä  "adieresis"
	0xe5: '\u00e5', //  å  "aring"
	0xe6: '\u00e6', //  æ  "ae"
	0xe7: '\u00e7', //  ç  "ccedilla"
	0xe8: '\u00e8', //  è  "egrave"
	0xe9: '\u00e9', //  é  "eacute"
	0xea: '\u00ea', //  ê  "ecircumflex"
	0xeb: '\u00eb', //  ë  "edieresis"
	0xec: '\u00ec', //  ì  "igrave"
	0xed: '\u00ed', //  í  "iacute"
	0xee: '\u00ee', //  î  "icircumflex"
	0xef: '\u00ef', //  ï  "idieresis"
	0xf0: '\u00f0', //  ð  "eth"
	0xf1: '\u00f1', //  ñ  "ntilde"
	0xf2: '\u00f2', //  ò  "ograve"
	0xf3: '\u00f3', //  ó  "oacute"
	0xf4: '\u00f4', //  ô  "ocircumflex"
	0xf5: '\u00f5', //  õ  "otilde"
	0xf6: '\u00f6', //  ö  "odieresis"
	0xf7: '\u00f7', //  ÷  "divide"
	0xf8: '\u00f8', //  ø  "oslash"
	0xf9: '\u00f9', //  ù  "ugrave"
	0xfa: '\u00fa', //  ú  "uacute"
	0xfb: '\u00fb', //  û  "ucircumflex"
	0xfc: '\u00fc', //  ü  "udieresis"
	0xfd: '\u00fd', //  ý  "yacute"
	0xfe: '\u00fe', //  þ  "thorn"
	0xff: '\u00ff', //  ÿ  "ydieresis"
}

var pdfdocEncodingRuneMap map[rune]byte

func init() {
	pdfdocEncodingRuneMap = map[rune]byte{}
	for b, r := range pdfDocEncoding {
		pdfdocEncodingRuneMap[r] = b
	}
}

// PDFDocEncodingToString decodes PDFDocEncoded byte slice `b` to unicode string.
func PDFDocEncodingToString(b []byte) string {
	var runes []rune
	for _, bval := range b {
		rune, has := pdfDocEncoding[bval]
		if !has {
			continue
		}

		runes = append(runes, rune)
	}

	return string(runes)
}

// StringToPDFDocEncoding encode go string `s` to PdfDocEncoding.
func StringToPDFDocEncoding(s string) []byte {
	var buf bytes.Buffer
	for _, r := range s {
		b, has := pdfdocEncodingRuneMap[r]
		if !has {
			continue
		}
		buf.WriteByte(b)
	}

	return buf.Bytes()
}
