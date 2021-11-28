package parser

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io/ioutil"
	"math/rand"
	"os"
	"reflect"
	"testing"

	"github.com/benoitkugler/pdf/contentstream"
	"github.com/benoitkugler/pdf/fonts"
	"github.com/benoitkugler/pdf/model"
)

var ops = [...]contentstream.Operation{
	// inline image data has its own test
	//   contentstream.OpBeginImage{},
	//   contentstream.OpEndImage{},
	//   contentstream.OpImageData{},
	contentstream.OpMoveSetShowText{},
	contentstream.OpMoveShowText{},
	contentstream.OpFillStroke{},
	contentstream.OpEOFillStroke{},
	contentstream.OpBeginMarkedContent{},
	contentstream.OpBeginMarkedContent{},
	contentstream.OpBeginMarkedContent{
		Properties: contentstream.PropertyListName("dsdd"),
	},
	contentstream.OpBeginMarkedContent{
		Properties: contentstream.PropertyListDict{"slmkds": Array{}},
	},
	contentstream.OpBeginText{},
	contentstream.OpBeginIgnoreUndef{},
	contentstream.OpSetStrokeColorSpace{},
	contentstream.OpMarkPoint{},
	contentstream.OpMarkPoint{
		Properties: contentstream.PropertyListName("dsdd"),
	},
	contentstream.OpMarkPoint{
		Properties: contentstream.PropertyListDict{"slmkds": Array{}},
	},
	contentstream.OpXObject{},
	contentstream.OpEndMarkedContent{},
	contentstream.OpEndText{},
	contentstream.OpEndIgnoreUndef{},
	contentstream.OpFill{},
	contentstream.OpSetStrokeGray{},
	contentstream.OpSetLineCap{},
	contentstream.OpSetStrokeCMYKColor{},
	contentstream.OpSetMiterLimit{},
	contentstream.OpMarkPoint{},
	contentstream.OpRestore{},
	contentstream.OpSetStrokeRGBColor{},
	contentstream.OpStroke{},
	contentstream.OpSetStrokeColor{},
	contentstream.OpSetStrokeColorN{
		Color: []float64{4, 5, 6},
	},
	contentstream.OpTextNextLine{},
	contentstream.OpTextMoveSet{},
	contentstream.OpShowSpaceText{
		Texts: []fonts.TextSpaced{
			{CharCodes: []byte("msdùlùd"), SpaceSubtractedAfter: 12},
			{CharCodes: []byte("AB"), SpaceSubtractedAfter: -5},
			{CharCodes: []byte("c"), SpaceSubtractedAfter: 2},
			{CharCodes: []byte("('')\\"), SpaceSubtractedAfter: 0},
		},
	},
	contentstream.OpSetTextLeading{},
	contentstream.OpSetCharSpacing{},
	contentstream.OpTextMove{},
	contentstream.OpSetFont{},
	contentstream.OpShowText{},
	contentstream.OpSetTextMatrix{},
	contentstream.OpSetTextRender{},
	contentstream.OpSetTextRise{},
	contentstream.OpSetWordSpacing{},
	contentstream.OpSetHorizScaling{},
	contentstream.OpClip{},
	contentstream.OpEOClip{},
	contentstream.OpCloseFillStroke{},
	contentstream.OpCloseEOFillStroke{},
	contentstream.OpCurveTo{},
	contentstream.OpConcat{},
	contentstream.OpSetFillColorSpace{},
	contentstream.OpSetDash{},
	contentstream.OpSetCharWidth{},
	contentstream.OpSetCacheDevice{},
	contentstream.OpFill{},
	contentstream.OpEOFill{},
	contentstream.OpSetFillGray{},
	contentstream.OpSetExtGState{},
	contentstream.OpClosePath{},
	contentstream.OpSetFlat{},
	contentstream.OpSetLineJoin{},
	contentstream.OpSetFillCMYKColor{},
	contentstream.OpLineTo{},
	contentstream.OpMoveTo{},
	contentstream.OpEndPath{},
	contentstream.OpSave{},
	contentstream.OpRectangle{},
	contentstream.OpSetFillRGBColor{},
	contentstream.OpSetRenderingIntent{},
	contentstream.OpCloseStroke{},
	contentstream.OpSetFillColor{},
	contentstream.OpSetFillColorN{Pattern: "sese"},
	contentstream.OpShFill{},
	contentstream.OpCurveTo1{},
	contentstream.OpSetLineWidth{},
	contentstream.OpCurveTo{},
}

func randOp() contentstream.Operation {
	j := rand.Intn(len(ops))
	return ops[j]
}

func randOps(nops int) []contentstream.Operation {
	l := make([]contentstream.Operation, nops)
	for i := range l {
		l[i] = randOp()
	}
	return l
}

func TestParseContent(t *testing.T) {
	exp := randOps(5000)
	ct := contentstream.WriteOperations(exp...)
	ops, err := ParseContent(ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(exp) != len(ops) {
		t.Errorf("expected %d ops, got %d", len(exp), len(ops))
	}
	for i := range exp {
		if !reflect.DeepEqual(exp[i], ops[i]) {
			t.Errorf("expected %v got %v", exp[i], ops[i])
		}
	}

	for _, op := range ops {
		_, err := ParseContent(contentstream.WriteOperations(op), nil)
		if err != nil {
			t.Error(err)
		}
	}
}

func randOperands() string {
	chars := []rune("////////<<<<<<>>>>>>>(((())))[[[]]789423azertyuiophjklmvbn,;:mùp$*")
	out := make([]rune, 10)
	for i := range out {
		out[i] = chars[rand.Intn(len(chars))]
	}
	return string(out)
}

func TestRandomInvalid(t *testing.T) {
	for range [5000]int{} {
		// alternate valid OPS and garbage input
		var in bytes.Buffer
		for range [20]int{} {
			in.WriteString(randOperands())
			randOp().Add(&in)
		}
		_, err := ParseContent(in.Bytes(), nil)
		if err == nil {
			t.Fatal("expected error on random input")
		}
	}
}

func TestTextSpaced(t *testing.T) {
	input := []byte("[45. (A) 20 20. (B)]TJ")
	ops, err := ParseContent(input, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 1 {
		t.Error()
	}
	tj, ok := ops[0].(contentstream.OpShowSpaceText)
	if !ok {
		t.Error()
	}
	expected := contentstream.OpShowSpaceText{Texts: []fonts.TextSpaced{
		{CharCodes: nil, SpaceSubtractedAfter: 45},
		{CharCodes: []byte("A"), SpaceSubtractedAfter: 40},
		{CharCodes: []byte("B")},
	}}
	if !reflect.DeepEqual(tj, expected) {
		t.Errorf("expected %v got %v", expected, tj)
	}

	invalid := "[(A) /Name (B)]TJ"
	_, err = ParseContent([]byte(invalid), nil)
	if err == nil {
		t.Fatalf("expected error on invalid input %s", invalid)
	}
}

func TestNormalizeSpacedText(t *testing.T) {
	in := "[(AB) (CD) 4 6 (AB) 5]TJ"
	ops, err := ParseContent([]byte(in), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 1 {
		t.Fatal()
	}
	exp := contentstream.OpShowSpaceText{
		Texts: []fonts.TextSpaced{
			{CharCodes: []byte("ABCD"), SpaceSubtractedAfter: 10},
			{CharCodes: []byte("AB"), SpaceSubtractedAfter: 5},
		},
	}
	if !reflect.DeepEqual(ops[0], exp) {
		t.Errorf("expected merged %v got %v", exp, ops[0])
	}
}

func randJPEG(size int) ([]byte, error) {
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for i := range img.Pix {
		img.Pix[i] = uint8(rand.Int())
	}
	err := jpeg.Encode(&buf, img, nil)
	return buf.Bytes(), err
}

func createImageStream(fi model.Name) (contentstream.OpBeginImage, error) {
	l := model.Filters{{Name: fi, DecodeParms: map[string]int{"unusedint": 4}}}
	if fi == model.DCT {
		content, err := randJPEG(30)
		return contentstream.OpBeginImage{
			Image: model.Image{
				Stream: model.Stream{Filter: l, Content: content}, BitsPerComponent: 8,
				Height: 30, Width: 30,
			},
			ColorSpace: contentstream.ImageColorSpaceName{ColorSpaceName: model.ColorSpaceRGB},
		}, err
	}
	b, err := ioutil.ReadFile("filters/samples/" + string(fi) + "_30x30.bin")
	if err != nil {
		return contentstream.OpBeginImage{}, err
	}
	return contentstream.OpBeginImage{
		Image: model.Image{
			Stream: model.Stream{
				Content: b,
				Filter:  l,
			}, BitsPerComponent: 8,
			Height: 30, Width: 30,
		}, ColorSpace: contentstream.ImageColorSpaceName{ColorSpaceName: model.ColorSpaceGray},
	}, err
}

func TestInlineData(t *testing.T) {
	filtersName := []model.ObjName{
		model.ASCII85,
		model.ASCIIHex,
		model.Flate,
		model.LZW,
		model.RunLength,
		model.DCT,
	}
	for _, fi := range filtersName {
		st, err := createImageStream(fi)
		if err != nil {
			t.Fatal(err)
		}

		var content bytes.Buffer
		st.Add(&content)
		contentStream := append(content.Bytes(), " f s /DeviceRGB cs"...)
		ops, err := ParseContent(contentStream, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(ops) != 4 {
			t.Errorf("expected 4 operation, got %v", ops)
		}
		img, ok := ops[0].(contentstream.OpBeginImage)
		if !ok {
			t.Errorf("expected Image, got %v", ops[0])
		}
		if !bytes.Equal(img.Image.Content, st.Image.Content) {
			t.Error("failed to retrieve image data")
		}
	}
}

func TestForgePDFInlineData(t *testing.T) {
	// generate samples demonstrating inline data
	filtersName := []model.ObjName{
		model.ASCII85,
		model.ASCIIHex,
		model.Flate,
		model.LZW,
		model.RunLength,
		model.DCT,
	}
	var doc model.Document
	for _, fi := range filtersName {
		st, err := createImageStream(fi)
		if err != nil {
			t.Fatal(err)
		}

		contentStream := contentstream.WriteOperations(
			contentstream.OpSave{},
			contentstream.OpConcat{Matrix: model.Matrix{15, 0, 0, 15, 100, 100}},
			st,
			contentstream.OpRestore{},
			contentstream.OpRectangle{W: 50, H: 50},
			contentstream.OpSetFillGray{G: 0.8},
			contentstream.OpFill{},
		)

		// add one page per image
		doc.Catalog.Pages.Kids = append(doc.Catalog.Pages.Kids,
			&model.PageObject{
				MediaBox: &model.Rectangle{Llx: 0, Lly: 0, Urx: 200, Ury: 200},
				Contents: []model.ContentStream{{Stream: model.Stream{Content: contentStream}}},
			})
	}
	f, err := os.Create("test/inline_images.pdf")
	if err != nil {
		t.Fatal(err)
	}
	err = doc.Write(f, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHexText(t *testing.T) {
	op := contentstream.OpShowSpaceGlyph{
		Glyphs: []contentstream.SpacedGlyph{
			{GID: 78, SpaceSubtractedBefore: -9},
			{GID: 45, SpaceSubtractedAfter: 10},
			{GID: 25, SpaceSubtractedBefore: -12, SpaceSubtractedAfter: 10},
		},
	}
	content := contentstream.WriteOperations(op)
	out, err := ParseContent(content, nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(out)
}
