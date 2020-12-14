/*
Copyright 2020 The pdfcpu Authors.

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

package parser

import (
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/benoitkugler/pdf/contentstream"
	"github.com/benoitkugler/pdf/model"
)

func TestParseResources(t *testing.T) {
	s := `/CS0 cs/DeviceGray CS/Span<</ActualText <FEFF000900090009>>> BDC
	/a1 BMC/a2 MP /a3 /MC0 BDC/P0 scn/RelativeColorimetric ri/P1 SCN/GS0 gs[(Q[i,j]/2.)16.6(The/]maxi\)-)]TJ/CS1 CS/a4<</A<FEFF>>> BDC /a5 <</A<FEFF>>>
	BDC (0.5*\(1/8\)*64 or +/4.\))Tj/T1_0 1 Tf <00150015> Tj /Im5 Do/a5 << /A <FEFF> >> BDC/a6/MC1 DP /a7<<>>DP
	BI /IM true/W 1/CS/InlineCS/H 1/BPC 1 ID 7 EI 
	BI /IM true/W 1/CS [/Indexed /DeviceGray 5 ()]  /H 1/BPC 1 ID 7 EI  Q /Pattern cs/Span<</ActualText<FEFF0009>>> BDC/SH1 sh`

	want := model.NewResourcesDict()
	want.ColorSpace["CS0"] = nil
	want.ColorSpace["CS1"] = nil
	want.ColorSpace["InlineCS"] = nil
	want.ExtGState["GS0"] = nil
	want.Font["T1_0"] = nil
	want.Pattern["P0"] = nil
	want.Pattern["P1"] = nil
	want.Properties["MC0"] = model.PropertyList{}
	want.Properties["MC1"] = model.PropertyList{}
	want.Shading["SH1"] = nil
	want.XObject["Im5"] = nil

	got, err := ParseContentResources([]byte(s), model.ResourcesColorSpace{"InlineCS": model.ColorSpaceGray})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("want:\n%v\ngot:\n%v\n", want, got)
	}
}

func TestFail(t *testing.T) {
	for _, bad := range []string{
		"/Name BMC 4",
		"BI ID 78 EI",
	} {
		_, err := ParseContentResources([]byte(bad), nil)
		if err == nil {
			t.Error("expected error on invalid input")
		}
	}
}

func TestContent(t *testing.T) {
	b, err := ioutil.ReadFile("test/content.txt")
	if err != nil {
		t.Fatal(err)
	}
	ops, err := ParseContent(b, nil)
	if err != nil {
		t.Error(err)
	}

	b2 := contentstream.WriteOperations(ops...)
	ops2, err := ParseContent(b2, nil)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(ops, ops2) {
		if len(ops) != len(ops2) {
			t.Errorf("expected same length")
		}
		for i, o := range ops {
			if !reflect.DeepEqual(o, ops2[i]) {
				t.Errorf("differents values %v and %v", o, ops2[i])
			}
		}
		t.Error("expected idempotent parsing")
	}
}
