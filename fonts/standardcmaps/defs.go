// Adobe predefined ToUnicode cmaps
package standardcmaps

import (
	"github.com/benoitkugler/pdf/fonts/cmaps"
	"github.com/benoitkugler/pdf/model"
)

type t = cmaps.ToUnicodeTranslation
type a = cmaps.ToUnicodeArray
type p = cmaps.ToUnicodePair

var ToUnicodeCMaps = map[model.ObjName]cmaps.UnicodeCMap{
	"Adobe-CNS1-UCS2":   Adobe_CNS1_UCS2,
	"Adobe-GB1-UCS2":    Adobe_GB1_UCS2,
	"Adobe-Japan1-UCS2": Adobe_Japan1_UCS2,
	"Adobe-Korea1-UCS2": Adobe_Korea1_UCS2,
	"Adobe-KR-UCS2":     Adobe_KR_UCS2,
}
