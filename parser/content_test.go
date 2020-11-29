package parser

import (
	"bytes"
	"math/rand"
	"reflect"
	"testing"

	"github.com/benoitkugler/pdf/contents"
	"github.com/benoitkugler/pdf/model"
)

var ops = [...]contents.Operation{
	//   contents.OpMoveSetShowText{},
	contents.OpMoveShowText{},
	//    contents.OpFillStroke{},
	// B  contents.OpEOFillStroke{},
	contents.OpBeginMarkedContent{},
	//   contents.OpBeginImage{},
	contents.OpBeginMarkedContent{},
	contents.OpBeginText{},
	//   contents.OpBeginIgnoreUndef{},
	contents.OpSetStrokeColorSpace{},
	contents.OpMarkPoint{},
	contents.OpXObject{},
	//   contents.OpEndImage{},
	contents.OpEndMarkedContent{},
	contents.OpEndText{},
	//   contents.OpEndIgnoreUndef{},
	//    contents.OpFill{},
	//    contents.OpSetStrokeGray{},
	//   contents.OpImageData{},
	//    contents.OpSetLineCap{},
	//    contents.OpSetStrokeCMYKColor{},
	//    contents.OpSetMiterLimit{},
	contents.OpMarkPoint{},
	contents.OpRestore{},
	contents.OpSetStrokeRGBColor{},
	contents.OpStroke{},
	//   contents.OpSetStrokeColor{},
	//  contents.OpSetStrokeColorN{},
	// T  contents.OpTextNextLine{},
	//   contents.OpTextMoveSet{},
	//   contents.OpShowSpaceText{},
	contents.OpSetTextLeading{},
	//   contents.OpSetCharSpacing{},
	contents.OpTextMove{},
	contents.OpSetFont{},
	contents.OpShowText{},
	contents.OpSetTextMatrix{},
	//   contents.OpSetTextRender{},
	//   contents.OpSetTextRise{},
	//   contents.OpSetWordSpacing{},
	//   contents.OpSetHorizScaling{},
	contents.OpClip{},
	// W  contents.OpEOClip{},
	//    contents.OpCloseFillStroke{},
	// b  contents.OpCloseEOFillStroke{},
	//    contents.OpCurveTo{},
	//   contents.OpConcat{},
	contents.OpSetFillColorSpace{},
	contents.OpSetDash{},
	//   contents.OpSetCharWidth{},
	//   contents.OpSetCacheDevice{},
	contents.OpFill{},
	// f  contents.OpEOFill{},
	contents.OpSetFillGray{},
	contents.OpSetExtGState{},
	//    contents.OpClosePath{},
	//    contents.OpSetFlat{},
	//    contents.OpSetLineJoin{},
	//    contents.OpSetFillCMYKColor{},
	contents.OpLineTo{},
	contents.OpMoveTo{},
	contents.OpEndPath{},
	contents.OpSave{},
	contents.OpRectangle{},
	contents.OpSetFillRGBColor{},
	contents.OpSetRenderingIntent{},
	//    contents.OpCloseStroke{},
	contents.OpSetFillColor{},
	contents.OpSetFillColorN{Pattern: "sese"},
	contents.OpShFill{},
	//    contents.OpCurveTo1{},
	contents.OpSetLineWidth{},
	//    contents.OpCurveTo{},
}

func randOp() contents.Operation {
	j := rand.Intn(len(ops))
	return ops[j]
}

func randOps(nops int) []contents.Operation {
	l := make([]contents.Operation, nops)
	for i := range l {
		l[i] = randOp()
	}
	return l
}

func TestParseContent(t *testing.T) {
	exp := randOps(5000)
	ct := contents.WriteOperations(exp...)
	ops, err := ParseContent(ct, model.ResourcesDict{})
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
}

func randOperands() string {
	chars := []rune("////////<<<<<<>>>>>>>(((())))[[[]]789423azertyuiophjklmvbn,;:mùp$*")
	out := make([]rune, 10)
	for i := range out {
		out[i] = chars[rand.Intn(len(chars))]
	}
	return string(out)
}

func TestRandom(t *testing.T) {
	for range [1000]int{} {
		// alternate valid OPS and garbage input
		var in bytes.Buffer
		for range [300]int{} {
			in.WriteString(randOperands())
			randOp().Add(&in)
		}
		_, err := ParseContent(in.Bytes(), model.ResourcesDict{})
		if err == nil {
			t.Fatal("expected error on random input")
		}
	}
}

func TestInlineData(t *testing.T) {
	filtersName := []model.Name{
		model.ASCII85,
		model.Flate,
		// model.ASCIIHex,
		// model.LZW,
		// model.RunLength,
	}
	for _, fi := range filtersName {
		in := make([]byte, 20)
		rand.Read(in)
		st, err := model.NewStream(in, model.Filters{{Name: fi}})
		if err != nil {
			t.Fatal(err)
		}

		contentStream := []byte("BI " + st.PDFCommonFields(false) + " ID ")
		contentStream = append(contentStream, st.Content...)
		contentStream = append(contentStream, "EI"...)

		ops, err := ParseContent(contentStream, model.ResourcesDict{})
		if err != nil {
			t.Fatal(err)
		}
		if len(ops) != 1 {
			t.Errorf("expected one operation, got %v", ops)
		}
		img, ok := ops[0].(contents.OpBeginImage)
		if !ok {
			t.Errorf("expected Image, got %v", ops[0])
		}
		if !bytes.Equal(img.Image.Content, st.Content) {
			t.Error("failed to retrieve image data")
		}
	}
}
