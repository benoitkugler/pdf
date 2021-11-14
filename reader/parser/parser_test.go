package parser

import (
	"bytes"
	"reflect"
	"testing"

	tokenizer "github.com/benoitkugler/pstokenizer"
)

func doTestParseObjectOK(parseString string, t *testing.T) {
	o, err := ParseObject([]byte(parseString))
	if err != nil {
		t.Errorf("ParseObject from byte slice failed: <%v>\n%s", err, parseString)
		return
	}
	_ = o.Write(nil, 0)

	pr := NewParserFromTokenizer(tokenizer.NewTokenizerFromReader(bytes.NewReader([]byte(parseString))))
	o2, err := pr.ParseObject()
	if err != nil {
		t.Errorf("ParseObject from reader failed: <%v>\n%s", err, parseString)
		return
	}
	if !reflect.DeepEqual(o, o2) {
		t.Errorf("expected same results, got %v and %v", o, o2)
	}
}

func doTestParseObjectFail(parseString string, t *testing.T) {
	_, err := ParseObject([]byte(parseString))
	if err == nil {
		t.Errorf("ParseObjectshould have returned an error for %s", parseString)
	}
}

func TestInitMethods(t *testing.T) {
	input := []byte("<</Key1/Value1/key1/Value2>>")

	o1, err := ParseObject(input)
	if err != nil {
		t.Fatal(err)
	}

	tk := tokenizer.NewTokenizer(input)
	p := NewParserFromTokenizer(tk)
	o2, err := p.ParseObject()
	if err != nil {
		t.Fatal(err)
	}

	p2 := NewParser(input)
	o3, err := p2.ParseObject()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(o1, o2) {
		t.Errorf("expected equals %v and %v", o1, o2)
	}
	if !reflect.DeepEqual(o2, o3) {
		t.Errorf("expected equals %v and %v", o2, o3)
	}
}

func TestParseDef(t *testing.T) {
	o, g, obj, err := ParseObjectDefinition([]byte("12 5 obj \n << /Type 3>>"), false)
	if err != nil {
		t.Fatal(err)
	}
	if o != 12 || g != 5 {
		t.Errorf("expected 12 5, got %d %d", o, g)
	}
	if exp := (Dict{"Type": Integer(3)}); !reflect.DeepEqual(obj, exp) {
		t.Errorf("expected %v got %v", exp, obj)
	}
	_, _, obj, err = ParseObjectDefinition([]byte("12 5 obj \n << /Type 3>>"), true)
	if err != nil {
		t.Fatal(err)
	}
	if obj != nil {
		t.Errorf("expected nil object, got %v", obj)
	}

	expectedError := [...]string{
		"12 5 ",
		"12  ",
		"<<>>  ",
		"12 5 obj <<  ",
		"< 5 obj <<  ",
		"5 < obj <<  ",
		"5 12 < <<  ",
	}
	for _, data := range expectedError {
		_, _, _, err = ParseObjectDefinition([]byte(data), false)
		if err == nil {
			t.Fatal("expected error")
		}
	}
}

func TestParseCommand(t *testing.T) {
	p := NewParser([]byte("ID stream 4"))
	p.ContentStreamMode = true
	o, err := p.ParseObject()
	if err != nil {
		t.Fatal(err)
	}
	_ = o.Write(nil, 0)
}

var datas = []string{
	"null      ",
	"true     ",
	"[true%comment\x0Anull]",
	"[[%comment\x0dnull][%empty\x0A\x0Dtrue]false%comment\x0A]",
	"<<>>",
	"<</Key %comment\x0a true%comment       \x0a\x0d>>",
	"<</Key/Value>>",
	"<</Key[/Val1/Val2\x0d%fopher\x0atrue]>>",
	"[<</k1[/name1]>><</k1[false true null]>>]",
	"/Name ",
	"/Na#20me",
	"[null]abc",
	"/(",
	"//",
	"/abc/",
	"/abc",
	"/abc/def",
	"%comment\x0D<c0c>%\x0a",
	"[<0ab>%comment\x0a]",
	"<</Key1<abc>/Key2<def>>>",
	"<< /Key1 <abc> /Key2 <def> >>",
	"<</Key1<AB>>>",
	"<</Key1<ABC>>>",
	"<</Key1<0ab>>>",
	"<</Key<>>>",
	"()",
	"(gopher\\\x28go)",
	"(gop\x0aher\\(go)",
	"(go\\pher\\)go)",
	"[%comment\x0d(gopher\\ago)%comment\x0a]",
	"()",
	"<</K(gopher)>>",
	"[(abc)true/n1<20f>]..",
	"[(abc)()]..",
	"[<743EEC2AFD93A438D87F5ED3D51166A8><B7FFF0ADB814244ABD8576D07849BE54>]",
	"1",
	"1/",
	"3.43",
	"3.43<",
	"1.2",
	"[<0ab>]",
	"1 0 R%comment\x0a",
	"[1 0 R /n 2 0 R]",
	"<</n 1 0 R>>",
	"(!\\(S:\\356[\\272H\\355>>R{sb\\007)",

	"<</Type /Pages /Count 24 /Kids [6 0 R 16 0 R 21 0 R 27 0 R 30 0 R 32 0 R 34 0 R 36 0 R 38 0 R 40 0 R 42 0 R 44 0 R 46 0 R 48 0 R 50 0 R 52 0 R 54 0 R 56 0 R 58 0 R 60 0 R 62 0 R 64 0 R 69 0 R 71 0 R] /MediaBox [0 0 595.2756 841.8898]>>",
	"<< /Key1 <abc> /Key2 <d> >>",
	"<<>>",
	"<<     >>",
	"<</Key1/Value1/key1/Value2>>",
	"<</Type/Page/Parent 2 0 R/Resources<</Font<</F1 5 0 R/F2 7 0 R/F3 9 0 R>>/XObject<</Image11 11 0 R>>/ProcSet[/PDF/Text/ImageB/ImageC/ImageI]>>/MediaBox[ 0 0 595.32 841.92]/Contents 4 0 R/Group<</Type/Group/S/Transparency/CS/DeviceRGB>>/Tabs/S/StructParents 0>>",
	"<</S/A>>",
	"<</K1 / /K2 /Name2>>",
	"<</Key/Value>>",
	"<< /Key	/Value>>",
	"<<	/Key/Value	>>",
	"<<	/Key	/Value	>>",
	"<</Key1/Value1/Key2/Value2>>",
	"<</Key1(abc)/Key2(def)>>..",
	"<</Key1(abc(inner1<<>>inner2)def)    >>..",
	"<</Key1<abc>/Key2<def>>>",
	"<< /Key1 <abc> /Key2 <def> >>",
	"<</Key1<AB>>>",
	"<</Key1<ABC>>>",
	"<</Key1<0ab>>>",
	"<</Key<>>>",
	"<< /Panose <01 05 02 02 03 00 00 00 00 00 00 00> >>",
	"<< /Panose < 0 0 2 6 6 6 5 6 5 2 2 4> >>",
	"<</Key <FEFF ABC2>>>",
	"<</Key<</Sub1 1/Sub2 2>>>>",
	"<</Key<</Sub1(xyz)>>>>",
	"<</Key<</Sub1[]>>>>",
	"<</Key<</Sub1[1]>>>>",
	"<</Key<</Sub1[(Go)]>>>>",
	"<</Key<</Sub1[(Go)]/Sub2[(rocks!)]>>>>",
	"<</A[/B1 /B2<</C 1>>]>>",
	"<</A[/B1 /B2<</C 1>> /B3]>>",
	"<</Name1[/CalRGB<</Matrix[0.41239 0.21264]/Gamma[2.22 2.22 2.22]/WhitePoint[0.95043 1 1.09]>>]>>",
	"<</A[/DictName<</A 123 /B<c0ff>>>]>>",
	"<</A[/B]>>",
	"<</Key1[<abc><def>12.24 (gopher)]>>",
	"<</Key1[<abc><def>12.24 (gopher)] /Key2[(abc)2.34[<c012>2 0 R]]>>",
	"<</Key1[1 2 3 [4]]>>",
	"<</K[<</Obj 71 0 R/Type/OBJR>>269 0 R]/P 258 0 R/S/Link/Pg 19 0 R>>",
	"<</Key1 true>>",
	"<</Key1 			false>>",
	"<</Key1 null /Key2 true /Key3 false>>",
	"<</Key1 16>>",
	"<</Key1 .034>>",
	"<</Key1 32 0 R>>",
	"<</Key1 32 0 R/Key2 32 /Key3 3.34>>",

	// doTestParseArrayOK("["+s+s+s+s1+"]",
	"[]",
	"[     ]",
	"[  ]abc",
	"[<AB >]",
	"[<     AB >]",
	"[<abc><def>]",
	"[<AB>]",
	"[<ABC>]",
	"[<0ab> <bcf098>]",
	"[<0ab>]",
	"[<0abc>]",
	"[<0abc> <345bca> <aaf>]",
	"[<01 05 02 02 03 00 00 00 00 00 00 00>]",
	"[< 0 0 2 6 6 6 5 6 5 2 2 4>]",
	"[(abc) (def) <20ff>]..",
	"[(abc)()]..",
	"[<743EEC2AFD93A438D87F5ED3D51166A8><B7FFF0ADB814244ABD8576D07849BE54>]",
	"[(abc(i)) <FFb> (<>)]",
	"[(abc)<20ff>]..",
	"[/N[/A/B/C]]",
	"[<</Obj 71 0 R/Type/OBJR>>269 0 R]",
	"[/Name 123<</A 123 /B<c0ff>>>]",
	"[/DictName<</A 123 /B<c0ff>>>]",
	"[/DictName<</A 123 /B<c0ff>/C[(Go!)]>>]",
	"[/Name<<>>]",
	"[/Name<</Sub 1>>]",
	"[/Name<</Sub<BBB>>>]",
	"[/Name<</Sub[1]>>]",
	"[/CalRGB<</Matrix[0.41239 0.21264]/Gamma[2.22 2.22 2.22]/WhitePoint[0.95043 1 1.09]>>]",
	"[/]",
	"[/ ]",
	"[/N]",
	"[/Name]",
	"[/First /Last]",
	"[ /First/Last]",
	"[ /First/Last  ]",
	"[/PDF/ImageC/Text]",
	"[/PDF /Text /ImageB /ImageC /ImageI]",
	"[<004>]/Name[(abc)]]",
	"[0.000000-16763662]",
	"[1.09 2.00056]",
	"[1.09 null true false [/Name1 2.00056]]",
	"[[2.22 2.22	2.22][0.95043 1 1.09]]",
	"[1]",
	"[1.01]",
	"[1 null]",
	"[1 true[2 0]]",
	"[1.09 2.00056]]",
	"[0 0 841.89 595.28]",
	"[ 487 190]",
	"[0.95043 1 1.09]",
	"[1 true[2 0 R 14 0 R]]",
	"[22 0 1 R 14 23 0 R 24 0 R 25 0 R 27 0 R 28 0 R 29 0 R 31 0 R]",
	"[/Name<<>>]",
	"[/Name<< /Sub /Value>>]",
	"[/Name<</Sub 1>>]",
	"[/Name<</Sub<BBB>>>]",
	"[/Name<</Sub[1]>>]",
	"[/CalRGB<</Matrix[0.41239 0.21264]/Gamma[2.22 2.22 2.22]/WhitePoint[0.95043 1 1.09]>>]",
}

// func BenchmarkParseOnePass(b *testing.B) {
// 	for i := 0; i < b.N; i++ {
// 		for _, data := range datas {
// 			_, _ = model.ObjParseOneObject(data)
// 		}
// 	}
// }

func BenchmarkParseTokenizer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, data := range datas {
			_, _ = ParseObject([]byte(data))
		}
	}
}
