/*
 * This file is subject to the terms and conditions defined in
 * file 'LICENSE.md', which is part of this source code package.
 */

package cmap

import "github.com/benoitkugler/pdf/model"

type cmapObject interface {
}

// type cmapName struct {
// 	Name string
// }

type cmapOperand string

// cmapHexString represents a PostScript hex string such as <FFFF>
type cmapHexString []byte

// type cmapFloat struct {
// 	val float64
// }

// type cmapInt struct {
// 	val int64
// }

// type cmapString struct {
// 	String string
// }

type cmapArray = []cmapObject

type cmapDict = map[model.Name]cmapObject
