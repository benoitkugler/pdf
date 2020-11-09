package writer

import (
	"bytes"
	"fmt"

	"github.com/benoitkugler/pdf/model"
)

// do not register in the cache
func (w *writer) writeFunction(f model.Function) ref {
	baseArgs := fmt.Sprintf("/Domain %s", writeRangeArray(f.Domain))
	if len(f.Range) != 0 {
		baseArgs += fmt.Sprintf(" /Range %s", writeRangeArray(f.Range))
	}
	var content []byte
	switch ft := f.FunctionType.(type) {
	case model.SampledFunction:
		content = writeSampledFunction(ft, baseArgs)
	case model.ExpInterpolationFunction:
		content = writeExpInterpolationFunction(ft, baseArgs)
	case model.StitchingFunction:
		// start by writing the "child" functions
		refs := make([]ref, len(ft.Functions))
		for i, f := range ft.Functions {
			refs[i] = w.writeFunction(f)
		}
		content = writeStitchingFunction(ft, baseArgs, refs)
	case model.PostScriptCalculatorFunction:
		content = writePostScriptCalculatorFunction(ft, baseArgs)
	}

	ref := w.defineObject(content)
	return ref
}

// return a reference to `f` (if needed, write a new object)
// panic with nil entry
func (w *writer) registerFunction(f *model.Function) ref {
	if ref, has := w.functions[f]; has {
		return ref
	}
	ref := w.writeFunction(*f)
	w.functions[f] = ref
	return ref
}

func writeSampledFunction(f model.SampledFunction, baseArgs string) []byte {
	var b bytes.Buffer
	b.WriteString("<< /FunctionType 0 ")
	b.WriteString(baseArgs)
	b.WriteString(f.ContentStream.PDFCommonFields())
	b.WriteString(fmt.Sprintf(" /Size %s /BitsPerSample %d", writeIntArray(f.Size), f.BitsPerSample))
	if f.Order != 0 {
		b.WriteString(fmt.Sprintf(" /Order %d", f.Order))
	}
	if len(f.Encode) != 0 {
		b.WriteString(" /Encode [ ")
		for _, v := range f.Encode {
			b.WriteString(fmt.Sprintf("%.3f %.3f ", v[0], v[1]))
		}
		b.WriteByte(']')
	}
	if len(f.Decode) != 0 {
		b.WriteString(" /Decode ")
		b.WriteString(writeRangeArray(f.Decode))
	}
	b.WriteString(" >>\n")
	b.WriteString("stream\n")
	b.Write(f.Content)
	b.WriteString("\nendstream")
	return b.Bytes()
}

func writeExpInterpolationFunction(f model.ExpInterpolationFunction, baseArgs string) []byte {
	c0, c1 := "", ""
	if len(f.C0) != 0 {
		c0 = " /C0 " + writeFloatArray(f.C0)
	}
	if len(f.C1) != 0 {
		c1 = " /C1 " + writeFloatArray(f.C1)
	}
	return []byte(fmt.Sprintf("<</FunctionType 2 %s%s%s /N %d>>", baseArgs, c0, c1, f.N))
}

func writeStitchingFunction(f model.StitchingFunction, baseArgs string, fnRefs []ref) []byte {
	return []byte(fmt.Sprintf("<</FunctionType 3 %s /Functions %s /Bounds %s /Encode %s>>",
		baseArgs, writeRefArray(fnRefs), writeFloatArray(f.Bounds), writePointArray(f.Encode)))
}

func writePostScriptCalculatorFunction(f model.PostScriptCalculatorFunction, baseArgs string) []byte {
	s := model.ContentStream(f).PDFCommonFields()
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("<</FunctionType 4 %s %s>>\n", baseArgs, s))
	b.WriteString("stream\n")
	b.Write(f.Content)
	b.WriteString("\nendstream")
	return b.Bytes()
}
