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

func parseObject(s string) error {
	tk := Tokenizer{data: []byte(s)}
	for token, err := tk.NextToken(); ; token, err = tk.NextToken() {
		// fmt.Println(token)
		if err != nil {
			return err
		}
		if token.Kind == EOF {
			break
		}
	}
	return nil
}

func doTestParseObjectOK(parseString string, t *testing.T) {
	err := parseObject(parseString)
	if err != nil {
		t.Errorf("parseObject failed: <%v>\n%s", err, parseString)
		return
	}
}

func doTestParseObjectFail(tokenValid bool, parseString string, t *testing.T) {
	err := parseObject(parseString)
	if !tokenValid && err == nil {
		t.Errorf("parseObject should have returned an error for %s\n", parseString)
	}
}

func TestParseObject(t *testing.T) {
	doTestParseObjectOK("null      ", t)
	doTestParseObjectOK("true     ", t)
	doTestParseObjectOK("[true%comment\x0Anull]", t)
	doTestParseObjectOK("[[%comment\x0dnull][%empty\x0A\x0Dtrue]false%comment\x0A]", t)
	doTestParseObjectOK("<<>>", t)
	doTestParseObjectOK("<</Key %comment\x0a true%comment       \x0a\x0d>>", t)
	doTestParseObjectOK("<</Key/Value>>", t)
	doTestParseObjectOK("<</Key[/Val1/Val2\x0d%gopher\x0atrue]>>", t)
	doTestParseObjectOK("[<</k1[/name1]>><</k1[false true null]>>]", t)
	doTestParseObjectOK("/Name ", t)
	doTestParseObjectFail(false, "/Na#me", t)
	doTestParseObjectFail(false, "/Na#2me", t)
	doTestParseObjectOK("/Na#20me", t)
	doTestParseObjectOK("[null]abc", t)

	doTestParseObjectFail(true, "/", t)
	doTestParseObjectFail(false, "/(", t)
	doTestParseObjectOK("//", t)
	doTestParseObjectOK("/abc/", t)
	doTestParseObjectOK("/abc", t)
	doTestParseObjectOK("/abc/def", t)

	doTestParseObjectOK("%comment\x0D<c0c>%\x0a", t)
	doTestParseObjectOK("[<0ab>%comment\x0a]", t)
	doTestParseObjectOK("<</Key1<abc>/Key2<def>>>", t)
	doTestParseObjectOK("<< /Key1 <abc> /Key2 <def> >>", t)
	doTestParseObjectOK("<</Key1<AB>>>", t)
	doTestParseObjectOK("<</Key1<ABC>>>", t)
	doTestParseObjectOK("<</Key1<0ab>>>", t)
	doTestParseObjectOK("<</Key<>>>", t)
	doTestParseObjectFail(true, "<>", t)

	doTestParseObjectOK("()", t)
	doTestParseObjectOK("(gopher\\\x28go)", t)
	doTestParseObjectOK("(gop\x0aher\\(go)", t)
	doTestParseObjectOK("(go\\pher\\)go)", t)

	doTestParseObjectOK("[%comment\x0d(gopher\\ago)%comment\x0a]", t)
	doTestParseObjectOK("()", t)
	doTestParseObjectOK("<</K(gopher)>>", t)

	doTestParseObjectOK("[(abc)true/n1<20f>]..", t)
	doTestParseObjectOK("[(abc)()]..", t)
	doTestParseObjectOK("[<743EEC2AFD93A438D87F5ED3D51166A8><B7FFF0ADB814244ABD8576D07849BE54>]", t)

	doTestParseObjectOK("1", t)
	doTestParseObjectOK("1/", t)

	doTestParseObjectOK("3.43", t)
	doTestParseObjectFail(false, "3.43<", t)

	doTestParseObjectOK("1.2", t)
	doTestParseObjectOK("[<0ab>]", t)

	doTestParseObjectOK("1 0 R%comment\x0a", t)
	doTestParseObjectOK("[1 0 R /n 2 0 R]", t)
	doTestParseObjectOK("<</n 1 0 R>>", t)
	doTestParseObjectOK("(!\\(S:\\356[\\272H\\355>>R{sb\\007)", t)

	doTestParseObjectFail(false, "<2O>", t)
	doTestParseObjectOK("(\\n\\t\\r\\b\\f\\(\\) \\\n \\\r)", t)
	doTestParseObjectOK("(\\053 \\53 \\0539 \\5 )", t)
	doTestParseObjectOK("(\r\n)", t)
	doTestParseObjectOK("(\r8)", t)
	doTestParseObjectFail(false, "(\r", t)
	doTestParseObjectFail(false, "(\\", t)
}
