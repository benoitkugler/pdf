package simpleencodings

// ZapfDingbats is the ZapfDingbats encoding.
var ZapfDingbats = map[rune]byte{32: 0x20, 8594: 0xd5, 8596: 0xd6, 8597: 0xd7, 9312: 0xac, 9313: 0xad, 9314: 0xae, 9315: 0xaf, 9316: 0xb0, 9317: 0xb1, 9318: 0xb2, 9319: 0xb3, 9320: 0xb4, 9321: 0xb5, 9632: 0x6e, 9650: 0x73, 9660: 0x74, 9670: 0x75, 9679: 0x6c, 9687: 0x77, 9733: 0x48, 9742: 0x25, 9755: 0x2a, 9758: 0x2b, 9824: 0xab, 9827: 0xa8, 9829: 0xaa, 9830: 0xa9, 9985: 0x21, 9986: 0x22, 9987: 0x23, 9988: 0x24, 9990: 0x26, 9991: 0x27, 9992: 0x28, 9993: 0x29, 9996: 0x2c, 9997: 0x2d, 9998: 0x2e, 9999: 0x2f, 10000: 0x30, 10001: 0x31, 10002: 0x32, 10003: 0x33, 10004: 0x34, 10005: 0x35, 10006: 0x36, 10007: 0x37, 10008: 0x38, 10009: 0x39, 10010: 0x3a, 10011: 0x3b, 10012: 0x3c, 10013: 0x3d, 10014: 0x3e, 10015: 0x3f, 10016: 0x40, 10017: 0x41, 10018: 0x42, 10019: 0x43, 10020: 0x44, 10021: 0x45, 10022: 0x46, 10023: 0x47, 10025: 0x49, 10026: 0x4a, 10027: 0x4b, 10028: 0x4c, 10029: 0x4d, 10030: 0x4e, 10031: 0x4f, 10032: 0x50, 10033: 0x51, 10034: 0x52, 10035: 0x53, 10036: 0x54, 10037: 0x55, 10038: 0x56, 10039: 0x57, 10040: 0x58, 10041: 0x59, 10042: 0x5a, 10043: 0x5b, 10044: 0x5c, 10045: 0x5d, 10046: 0x5e, 10047: 0x5f, 10048: 0x60, 10049: 0x61, 10050: 0x62, 10051: 0x63, 10052: 0x64, 10053: 0x65, 10054: 0x66, 10055: 0x67, 10056: 0x68, 10057: 0x69, 10058: 0x6a, 10059: 0x6b, 10061: 0x6d, 10063: 0x6f, 10064: 0x70, 10065: 0x71, 10066: 0x72, 10070: 0x76, 10072: 0x78, 10073: 0x79, 10074: 0x7a, 10075: 0x7b, 10076: 0x7c, 10077: 0x7d, 10078: 0x7e, 10081: 0xa1, 10082: 0xa2, 10083: 0xa3, 10084: 0xa4, 10085: 0xa5, 10086: 0xa6, 10087: 0xa7, 10102: 0xb6, 10103: 0xb7, 10104: 0xb8, 10105: 0xb9, 10106: 0xba, 10107: 0xbb, 10108: 0xbc, 10109: 0xbd, 10110: 0xbe, 10111: 0xbf, 10112: 0xc0, 10113: 0xc1, 10114: 0xc2, 10115: 0xc3, 10116: 0xc4, 10117: 0xc5, 10118: 0xc6, 10119: 0xc7, 10120: 0xc8, 10121: 0xc9, 10122: 0xca, 10123: 0xcb, 10124: 0xcc, 10125: 0xcd, 10126: 0xce, 10127: 0xcf, 10128: 0xd0, 10129: 0xd1, 10130: 0xd2, 10131: 0xd3, 10132: 0xd4, 10136: 0xd8, 10137: 0xd9, 10138: 0xda, 10139: 0xdb, 10140: 0xdc, 10141: 0xdd, 10142: 0xde, 10143: 0xdf, 10144: 0xe0, 10145: 0xe1, 10146: 0xe2, 10147: 0xe3, 10148: 0xe4, 10149: 0xe5, 10150: 0xe6, 10151: 0xe7, 10152: 0xe8, 10153: 0xe9, 10154: 0xea, 10155: 0xeb, 10156: 0xec, 10157: 0xed, 10158: 0xee, 10159: 0xef, 10161: 0xf1, 10162: 0xf2, 10163: 0xf3, 10164: 0xf4, 10165: 0xf5, 10166: 0xf6, 10167: 0xf7, 10168: 0xf8, 10169: 0xf9, 10170: 0xfa, 10171: 0xfb, 10172: 0xfc, 10173: 0xfd, 10174: 0xfe, 63703: 0x80, 63704: 0x81, 63705: 0x82, 63706: 0x83, 63707: 0x84, 63708: 0x85, 63709: 0x86, 63710: 0x87, 63711: 0x88, 63712: 0x89, 63713: 0x8a, 63714: 0x8b, 63715: 0x8c, 63716: 0x8d}

// ZapfDingbatsNames is the ZapfDingbats encoding.
var ZapfDingbatsNames = [256]string{
	32:  "space", // U+0020 ' '
	33:  "a1",    // U+0021 '!'
	34:  "a2",    // U+0022 '"'
	35:  "a202",  // U+0023 '#'
	36:  "a3",    // U+0024 '$'
	37:  "a4",    // U+0025 '%'
	38:  "a5",    // U+0026 '&'
	39:  "a119",  // U+0027 '''
	40:  "a118",  // U+0028 '('
	41:  "a117",  // U+0029 ')'
	42:  "a11",   // U+002A '*'
	43:  "a12",   // U+002B '+'
	44:  "a13",   // U+002C ','
	45:  "a14",   // U+002D '-'
	46:  "a15",   // U+002E '.'
	47:  "a16",   // U+002F '/'
	48:  "a105",  // U+0030 '0'
	49:  "a17",   // U+0031 '1'
	50:  "a18",   // U+0032 '2'
	51:  "a19",   // U+0033 '3'
	52:  "a20",   // U+0034 '4'
	53:  "a21",   // U+0035 '5'
	54:  "a22",   // U+0036 '6'
	55:  "a23",   // U+0037 '7'
	56:  "a24",   // U+0038 '8'
	57:  "a25",   // U+0039 '9'
	58:  "a26",   // U+003A ':'
	59:  "a27",   // U+003B ';'
	60:  "a28",   // U+003C '<'
	61:  "a6",    // U+003D '='
	62:  "a7",    // U+003E '>'
	63:  "a8",    // U+003F '?'
	64:  "a9",    // U+0040 '@'
	65:  "a10",   // U+0041 'A'
	66:  "a29",   // U+0042 'B'
	67:  "a30",   // U+0043 'C'
	68:  "a31",   // U+0044 'D'
	69:  "a32",   // U+0045 'E'
	70:  "a33",   // U+0046 'F'
	71:  "a34",   // U+0047 'G'
	72:  "a35",   // U+0048 'H'
	73:  "a36",   // U+0049 'I'
	74:  "a37",   // U+004A 'J'
	75:  "a38",   // U+004B 'K'
	76:  "a39",   // U+004C 'L'
	77:  "a40",   // U+004D 'M'
	78:  "a41",   // U+004E 'N'
	79:  "a42",   // U+004F 'O'
	80:  "a43",   // U+0050 'P'
	81:  "a44",   // U+0051 'Q'
	82:  "a45",   // U+0052 'R'
	83:  "a46",   // U+0053 'S'
	84:  "a47",   // U+0054 'T'
	85:  "a48",   // U+0055 'U'
	86:  "a49",   // U+0056 'V'
	87:  "a50",   // U+0057 'W'
	88:  "a51",   // U+0058 'X'
	89:  "a52",   // U+0059 'Y'
	90:  "a53",   // U+005A 'Z'
	91:  "a54",   // U+005B '['
	92:  "a55",   // U+005C '\'
	93:  "a56",   // U+005D ']'
	94:  "a57",   // U+005E '^'
	95:  "a58",   // U+005F '_'
	96:  "a59",   // U+0060 '`'
	97:  "a60",   // U+0061 'a'
	98:  "a61",   // U+0062 'b'
	99:  "a62",   // U+0063 'c'
	100: "a63",   // U+0064 'd'
	101: "a64",   // U+0065 'e'
	102: "a65",   // U+0066 'f'
	103: "a66",   // U+0067 'g'
	104: "a67",   // U+0068 'h'
	105: "a68",   // U+0069 'i'
	106: "a69",   // U+006A 'j'
	107: "a70",   // U+006B 'k'
	108: "a71",   // U+006C 'l'
	109: "a72",   // U+006D 'm'
	110: "a73",   // U+006E 'n'
	111: "a74",   // U+006F 'o'
	112: "a203",  // U+0070 'p'
	113: "a75",   // U+0071 'q'
	114: "a204",  // U+0072 'r'
	115: "a76",   // U+0073 's'
	116: "a77",   // U+0074 't'
	117: "a78",   // U+0075 'u'
	118: "a79",   // U+0076 'v'
	119: "a81",   // U+0077 'w'
	120: "a82",   // U+0078 'x'
	121: "a83",   // U+0079 'y'
	122: "a84",   // U+007A 'z'
	123: "a97",   // U+007B '{'
	124: "a98",   // U+007C '|'
	125: "a99",   // U+007D '}'
	126: "a100",  // U+007E '~'
	161: "a101",  // U+00A1 '¡'
	162: "a102",  // U+00A2 '¢'
	163: "a103",  // U+00A3 '£'
	164: "a104",  // U+00A4 '¤'
	165: "a106",  // U+00A5 '¥'
	166: "a107",  // U+00A6 '¦'
	167: "a108",  // U+00A7 '§'
	168: "a112",  // U+00A8 '¨'
	169: "a111",  // U+00A9 '©'
	170: "a110",  // U+00AA 'ª'
	171: "a109",  // U+00AB '«'
	172: "a120",  // U+00AC '¬'
	173: "a121",  // U+00AD
	174: "a122",  // U+00AE '®'
	175: "a123",  // U+00AF '¯'
	176: "a124",  // U+00B0 '°'
	177: "a125",  // U+00B1 '±'
	178: "a126",  // U+00B2 '²'
	179: "a127",  // U+00B3 '³'
	180: "a128",  // U+00B4 '´'
	181: "a129",  // U+00B5 'µ'
	182: "a130",  // U+00B6 '¶'
	183: "a131",  // U+00B7 '·'
	184: "a132",  // U+00B8 '¸'
	185: "a133",  // U+00B9 '¹'
	186: "a134",  // U+00BA 'º'
	187: "a135",  // U+00BB '»'
	188: "a136",  // U+00BC '¼'
	189: "a137",  // U+00BD '½'
	190: "a138",  // U+00BE '¾'
	191: "a139",  // U+00BF '¿'
	192: "a140",  // U+00C0 'À'
	193: "a141",  // U+00C1 'Á'
	194: "a142",  // U+00C2 'Â'
	195: "a143",  // U+00C3 'Ã'
	196: "a144",  // U+00C4 'Ä'
	197: "a145",  // U+00C5 'Å'
	198: "a146",  // U+00C6 'Æ'
	199: "a147",  // U+00C7 'Ç'
	200: "a148",  // U+00C8 'È'
	201: "a149",  // U+00C9 'É'
	202: "a150",  // U+00CA 'Ê'
	203: "a151",  // U+00CB 'Ë'
	204: "a152",  // U+00CC 'Ì'
	205: "a153",  // U+00CD 'Í'
	206: "a154",  // U+00CE 'Î'
	207: "a155",  // U+00CF 'Ï'
	208: "a156",  // U+00D0 'Ð'
	209: "a157",  // U+00D1 'Ñ'
	210: "a158",  // U+00D2 'Ò'
	211: "a159",  // U+00D3 'Ó'
	212: "a160",  // U+00D4 'Ô'
	213: "a161",  // U+00D5 'Õ'
	214: "a163",  // U+00D6 'Ö'
	215: "a164",  // U+00D7 '×'
	216: "a196",  // U+00D8 'Ø'
	217: "a165",  // U+00D9 'Ù'
	218: "a192",  // U+00DA 'Ú'
	219: "a166",  // U+00DB 'Û'
	220: "a167",  // U+00DC 'Ü'
	221: "a168",  // U+00DD 'Ý'
	222: "a169",  // U+00DE 'Þ'
	223: "a170",  // U+00DF 'ß'
	224: "a171",  // U+00E0 'à'
	225: "a172",  // U+00E1 'á'
	226: "a173",  // U+00E2 'â'
	227: "a162",  // U+00E3 'ã'
	228: "a174",  // U+00E4 'ä'
	229: "a175",  // U+00E5 'å'
	230: "a176",  // U+00E6 'æ'
	231: "a177",  // U+00E7 'ç'
	232: "a178",  // U+00E8 'è'
	233: "a179",  // U+00E9 'é'
	234: "a193",  // U+00EA 'ê'
	235: "a180",  // U+00EB 'ë'
	236: "a199",  // U+00EC 'ì'
	237: "a181",  // U+00ED 'í'
	238: "a200",  // U+00EE 'î'
	239: "a182",  // U+00EF 'ï'
	241: "a201",  // U+00F1 'ñ'
	242: "a183",  // U+00F2 'ò'
	243: "a184",  // U+00F3 'ó'
	244: "a197",  // U+00F4 'ô'
	245: "a185",  // U+00F5 'õ'
	246: "a194",  // U+00F6 'ö'
	247: "a198",  // U+00F7 '÷'
	248: "a186",  // U+00F8 'ø'
	249: "a195",  // U+00F9 'ù'
	250: "a187",  // U+00FA 'ú'
	251: "a188",  // U+00FB 'û'
	252: "a189",  // U+00FC 'ü'
	253: "a190",  // U+00FD 'ý'
	254: "a191",  // U+00FE 'þ'
}
