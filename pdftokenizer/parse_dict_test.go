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

package pdftokenizer

import (
	"testing"
)

func doTestParseDictGeneral(t *testing.T) {
	doTestParseObjectOK("<</Type /Pages /Count 24 /Kids [6 0 R 16 0 R 21 0 R 27 0 R 30 0 R 32 0 R 34 0 R 36 0 R 38 0 R 40 0 R 42 0 R 44 0 R 46 0 R 48 0 R 50 0 R 52 0 R 54 0 R 56 0 R 58 0 R 60 0 R 62 0 R 64 0 R 69 0 R 71 0 R] /MediaBox [0 0 595.2756 841.8898]>>", t)
	doTestParseObjectOK("<< /Key1 <abc> /Key2 <d> >>", t)
	doTestParseObjectFail(true, "<<", t)
	doTestParseObjectFail(false, "<<>", t)
	doTestParseObjectOK("<<>>", t)
	doTestParseObjectOK("<<     >>", t)
	doTestParseObjectOK("<</Key1/Value1/key1/Value2>>", t)
	doTestParseObjectOK("<</Type/Page/Parent 2 0 R/Resources<</Font<</F1 5 0 R/F2 7 0 R/F3 9 0 R>>/XObject<</Image11 11 0 R>>/ProcSet[/PDF/Text/ImageB/ImageC/ImageI]>>/MediaBox[ 0 0 595.32 841.92]/Contents 4 0 R/Group<</Type/Group/S/Transparency/CS/DeviceRGB>>/Tabs/S/StructParents 0>>", t)
}

func doTestParseDictNameObjects(t *testing.T) {
	// Name Objects
	doTestParseObjectOK("<</S/A>>", t) // empty name
	doTestParseObjectOK("<</K1 / /K2 /Name2>>", t)
	doTestParseObjectOK("<</Key/Value>>", t)
	doTestParseObjectOK("<< /Key	/Value>>", t)
	doTestParseObjectOK("<<	/Key/Value	>>", t)
	doTestParseObjectOK("<<	/Key	/Value	>>", t)
	doTestParseObjectOK("<</Key1/Value1/Key2/Value2>>", t)
}

func doTestParseDictStringLiteral(t *testing.T) {
	// String literals
	doTestParseObjectOK("<</Key1(abc)/Key2(def)>>..", t)
	doTestParseObjectOK("<</Key1(abc(inner1<<>>inner2)def)    >>..", t)
}

func TestBug(t *testing.T) {
	doTestParseObjectOK("[/Name 123<</A 123 /B<c0ff>>>]", t)
}

func doTestParseDictHexLiteral(t *testing.T) {
	// Hex literals
	doTestParseObjectFail(false, "<</Key<>>", t)
	doTestParseObjectFail(false, "<</Key<a4>>", t)
	doTestParseObjectFail(true, "<</Key<    >", t)
	doTestParseObjectFail(true, "<</Key<ade>", t)
	doTestParseObjectFail(false, "<</Key<ABG>>>", t)
	doTestParseObjectFail(false, "<</Key<   ABG>>>", t)
	doTestParseObjectFail(true, "<</Key<0ab><bcf098>", t)
	doTestParseObjectOK("<</Key1<abc>/Key2<def>>>", t)
	doTestParseObjectOK("<< /Key1 <abc> /Key2 <def> >>", t)
	doTestParseObjectOK("<</Key1<AB>>>", t)
	doTestParseObjectOK("<</Key1<ABC>>>", t)
	doTestParseObjectOK("<</Key1<0ab>>>", t)
	doTestParseObjectOK("<</Key<>>>", t)
	doTestParseObjectOK("<< /Panose <01 05 02 02 03 00 00 00 00 00 00 00> >>", t)
	doTestParseObjectOK("<< /Panose < 0 0 2 6 6 6 5 6 5 2 2 4> >>", t)
	doTestParseObjectOK("<</Key <FEFF ABC2>>>", t)
}

func doTestParseDictDict(t *testing.T) {
	// Dictionaries
	doTestParseObjectOK("<</Key<</Sub1 1/Sub2 2>>>>", t)
	doTestParseObjectOK("<</Key<</Sub1(xyz)>>>>", t)
	doTestParseObjectOK("<</Key<</Sub1[]>>>>", t)
	doTestParseObjectOK("<</Key<</Sub1[1]>>>>", t)
	doTestParseObjectOK("<</Key<</Sub1[(Go)]>>>>", t)
	doTestParseObjectOK("<</Key<</Sub1[(Go)]/Sub2[(rocks!)]>>>>", t)
	doTestParseObjectOK("<</A[/B1 /B2<</C 1>>]>>", t)
	doTestParseObjectOK("<</A[/B1 /B2<</C 1>> /B3]>>", t)
	doTestParseObjectOK("<</Name1[/CalRGB<</Matrix[0.41239 0.21264]/Gamma[2.22 2.22 2.22]/WhitePoint[0.95043 1 1.09]>>]>>", t)
	doTestParseObjectOK("<</A[/DictName<</A 123 /B<c0ff>>>]>>", t)
}

func doTestParseDictArray(t *testing.T) {
	// Arrays
	doTestParseObjectOK("<</A[/B]>>", t)
	doTestParseObjectOK("<</Key1[<abc><def>12.24 (gopher)]>>", t)
	doTestParseObjectOK("<</Key1[<abc><def>12.24 (gopher)] /Key2[(abc)2.34[<c012>2 0 R]]>>", t)
	doTestParseObjectOK("<</Key1[1 2 3 [4]]>>", t)
	doTestParseObjectOK("<</K[<</Obj 71 0 R/Type/OBJR>>269 0 R]/P 258 0 R/S/Link/Pg 19 0 R>>", t)
}

func doTestParseDictBool(t *testing.T) {
	// null, true, false
	doTestParseObjectOK("<</Key1 true>>", t)
	doTestParseObjectOK("<</Key1 			false>>", t)
	doTestParseObjectOK("<</Key1 null /Key2 true /Key3 false>>", t)
	doTestParseObjectFail(true, "<</Key1 TRUE>>", t)
}

func doTestParseDictNumerics(t *testing.T) {
	// Numerics
	doTestParseObjectOK("<</Key1 16>>", t)
	doTestParseObjectOK("<</Key1 .034>>", t)
	doTestParseObjectFail(true, "<</Key1 ,034>>", t)
}

func doTestParseDictIndirectRefs(t *testing.T) {
	// Indirect object references
	doTestParseObjectOK("<</Key1 32 0 R>>", t)
	doTestParseObjectOK("<</Key1 32 0 R/Key2 32 /Key3 3.34>>", t)
}

func TestParseDict(t *testing.T) {
	doTestParseDictGeneral(t)
	doTestParseDictNameObjects(t)
	doTestParseDictStringLiteral(t)
	doTestParseDictHexLiteral(t)
	doTestParseDictDict(t)
	doTestParseDictArray(t)
	doTestParseDictBool(t)
	doTestParseDictNumerics(t)
	doTestParseDictIndirectRefs(t)
}
