/*
Copyright 2018 The pdfcpu Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tokenizer

import (
	"strings"
	"testing"
)

func TestParseArray(t *testing.T) {
	s := strings.Repeat("null ", 100) // 10000 x null

	s1 := "15538 0 R 15538 0 R 15538 0 R 15538 0 R 15538 0 R 15538 0 R 15539 0 R 15539 0 R 15539 0 R 15539 0 R 15540 0 R 15540 0 R 15540 0 R 15541 0 R 15541 0 R 15541 0 R 15541 0 R 15541 0 R 15541 0 R null 15541 0 R 15541 0 R 15542 0 R 15542 0 R 17468 0 R 17469 0 R 15543 0 R 17466 0 R 17467 0 R 15545 0 R 15545 0 R 15545 0 R 15545 0 R 15545 0 R 15545 0 R 15546 0 R 15546 0 R 15546 0 R 15546 0 R 15546 0 R 15546 0 R 15547 0 R 15548 0 R"

	doTestParseObjectOK("["+s+s+s+s1+"]", t)

	doTestParseObjectOK("[]", t)
	doTestParseObjectOK("[     ]", t)
	doTestParseObjectOK("[  ]abc", t)

	//	Negative testing
	doTestParseObjectFail(true, "[", t)
	doTestParseObjectFail(true, "[            ", t)

	//	Hex literals
	doTestParseObjectFail(false, "[<", t)
	doTestParseObjectFail(true, "[<>", t)
	doTestParseObjectFail(true, "[<    >", t)
	doTestParseObjectFail(false, "[   <", t)
	doTestParseObjectFail(true, "[   <>", t)
	doTestParseObjectFail(true, "[   <    >", t)
	doTestParseObjectFail(false, "[<ABG>", t)
	doTestParseObjectFail(true, "[<FEFF ABC2>", t) // white space shall be ignored
	doTestParseObjectFail(false, "[<   ABG>", t)
	doTestParseObjectFail(true, "[<AB>", t)
	doTestParseObjectFail(true, "[<ABc>    ", t)
	doTestParseObjectFail(true, "[<0ab> <bcf098>", t)
	doTestParseObjectFail(true, "[a]", t)
	doTestParseObjectFail(true, "[<0ab> <bcf098>a]", t)
	doTestParseObjectFail(true, "[<0ab> <bcf098> a]", t)

	doTestParseObjectOK("[<AB >]", t)
	doTestParseObjectOK("[<     AB >]", t)
	doTestParseObjectOK("[<abc><def>]", t)
	doTestParseObjectOK("[<AB>]", t)
	doTestParseObjectOK("[<ABC>]", t)
	doTestParseObjectOK("[<0ab> <bcf098>]", t)
	doTestParseObjectOK("[<0ab>]", t)
	doTestParseObjectOK("[<0abc>]", t)
	doTestParseObjectOK("[<0abc> <345bca> <aaf>]", t)
	doTestParseObjectOK("[<01 05 02 02 03 00 00 00 00 00 00 00>]", t)
	doTestParseObjectOK("[< 0 0 2 6 6 6 5 6 5 2 2 4>]", t)

	// String literals
	doTestParseObjectOK("[(abc) (def) <20ff>]..", t)
	doTestParseObjectOK("[(abc)()]..", t)
	doTestParseObjectOK("[<743EEC2AFD93A438D87F5ED3D51166A8><B7FFF0ADB814244ABD8576D07849BE54>]", t)

	//	Mixed string and hex literals
	doTestParseObjectOK("[(abc(i)) <FFb> (<>)]", t)
	doTestParseObjectOK("[(abc)<20ff>]..", t)

	//	Arrays
	doTestParseObjectOK("[/N[/A/B/C]]", t)

	//	Dictionaries
	doTestParseObjectOK("[<</Obj 71 0 R/Type/OBJR>>269 0 R]", t)
	doTestParseObjectOK("[/Name 123<</A 123 /B<c0ff>>>]", t)

	doTestParseObjectOK("[/DictName<</A 123 /B<c0ff>>>]", t)
	doTestParseObjectOK("[/DictName<</A 123 /B<c0ff>/C[(Go!)]>>]", t)
	doTestParseObjectOK("[/Name<<>>]", t)
	doTestParseObjectOK("[/Name<</Sub 1>>]", t)
	doTestParseObjectOK("[/Name<</Sub<BBB>>>]", t)
	doTestParseObjectOK("[/Name<</Sub[1]>>]", t)
	doTestParseObjectOK("[/CalRGB<</Matrix[0.41239 0.21264]/Gamma[2.22 2.22 2.22]/WhitePoint[0.95043 1 1.09]>>]", t)

	// Name objects
	doTestParseObjectOK("[/]", t)
	doTestParseObjectOK("[/ ]", t)
	doTestParseObjectOK("[/N]", t)
	doTestParseObjectOK("[/Name]", t)
	doTestParseObjectOK("[/First /Last]", t)
	doTestParseObjectOK("[ /First/Last]", t)
	doTestParseObjectOK("[ /First/Last  ]", t)
	doTestParseObjectOK("[/PDF/ImageC/Text]", t)
	doTestParseObjectOK("[/PDF /Text /ImageB /ImageC /ImageI]", t)
	doTestParseObjectOK("[<004>]/Name[(abc)]]", t)

	//	Numerics
	doTestParseObjectOK("[0.000000-16763662]", t) // = -16763662
	doTestParseObjectOK("[1.09 2.00056]", t)
	doTestParseObjectOK("[1.09 null true false [/Name1 2.00056]]", t)
	doTestParseObjectOK("[[2.22 2.22	2.22][0.95043 1 1.09]]", t)
	doTestParseObjectOK("[1]", t)
	doTestParseObjectOK("[1.01]", t)
	doTestParseObjectOK("[1 null]", t)
	doTestParseObjectOK("[1 true[2 0]]", t)
	doTestParseObjectOK("[1.09 2.00056]]", t)
	doTestParseObjectOK("[0 0 841.89 595.28]", t)
	doTestParseObjectOK("[ 487 190]", t)
	doTestParseObjectOK("[0.95043 1 1.09]", t)

	// Indirect object references
	doTestParseObjectOK("[1 true[2 0 R 14 0 R]]", t)
	doTestParseObjectOK("[22 0 1 R 14 23 0 R 24 0 R 25 0 R 27 0 R 28 0 R 29 0 R 31 0 R]", t)

	// Dictionaries
	doTestParseObjectOK("[/Name<<>>]", t)
	doTestParseObjectOK("[/Name<< /Sub /Value>>]", t)
	doTestParseObjectOK("[/Name<</Sub 1>>]", t)
	doTestParseObjectOK("[/Name<</Sub<BBB>>>]", t)
	doTestParseObjectOK("[/Name<</Sub[1]>>]", t)
	doTestParseObjectOK("[/CalRGB<</Matrix[0.41239 0.21264]/Gamma[2.22 2.22 2.22]/WhitePoint[0.95043 1 1.09]>>]", t)
}
