/*
 * This file is subject to the terms and conditions defined in
 * file 'LICENSE.md', which is part of this source code package.
 */

package cmap

import (
	"unicode/utf16"
)

// hexToCharCode returns the integer that is encoded in `shex` as a big-endian hex value
func hexToCharCode(shex cmapHexString) CharCode {
	code := CharCode(0)
	for _, v := range shex {
		code <<= 8
		code |= CharCode(v)
	}
	return code
}

// hexToString decodes the UTF-16BE encoded string `shex` to unicode runes.
// 9.10.3 ToUnicode CMaps (page 293)
// • It shall use the beginbfchar, endbfchar, beginbfrange, and endbfrange operators to define the
// mapping from character codes to Unicode character sequences expressed in UTF-16BE encoding.
func hexToRunes(shex cmapHexString) []rune {
	if len(shex) == 1 {
		return []rune{rune(shex[0])}
	}
	b := shex
	if len(b)%2 != 0 {
		b = append(b, 0)
	}
	n := len(b) >> 1
	chars := make([]uint16, n)
	for i := 0; i < n; i++ {
		chars[i] = uint16(b[i<<1])<<8 + uint16(b[i<<1+1])
	}
	runes := utf16.Decode(chars)
	return runes
}

// hexToRune is the same as hexToRunes but expects only a single rune to be decoded.
func hexToRune(shex cmapHexString) rune {
	runes := hexToRunes(shex)
	if n := len(runes); n == 0 {
		return MissingCodeRune
	}
	return runes[0]
}

func min(i, j int) int {
	if i < j {
		return i
	}
	return j
}
