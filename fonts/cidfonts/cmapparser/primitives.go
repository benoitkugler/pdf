package cmapparser

import "github.com/benoitkugler/pdf/model"

type cmapObject interface {
}

type cmapOperand string

// cmapHexString represents a PostScript hex string such as <FFFF>
type cmapHexString []byte

type cmapArray = []cmapObject

type cmapDict = map[model.Name]cmapObject
