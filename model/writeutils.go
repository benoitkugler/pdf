package model

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func writeIntArray(as []int) string {
	b := make([]string, len(as))
	for i, a := range as {
		b[i] = strconv.Itoa(a)
	}
	return fmt.Sprintf("[%s]", strings.Join(b, " "))
}

func writeFloatArray(as []float64) string {
	b := make([]string, len(as))
	for i, a := range as {
		b[i] = fmt.Sprintf("%.3f", a)
	}
	return fmt.Sprintf("[%s]", strings.Join(b, " "))
}

func writeRefArray(as []reference) string {
	b := make([]string, len(as))
	for i, ref := range as {
		b[i] = ref.String()
	}
	return fmt.Sprintf("[%s]", strings.Join(b, " "))
}

func writePointArray(rs [][2]float64) string {
	b := make([]string, len(rs))
	for i, a := range rs {
		b[i] = fmt.Sprintf("%.3f %.3f ", a[0], a[1])
	}
	return fmt.Sprintf("[%s]", strings.Join(b, " "))
}

func writeRangeArray(rs []Range) string {
	b := make([]string, len(rs))
	for i, a := range rs {
		b[i] = fmt.Sprintf("%.3f %.3f ", a[0], a[1])
	}
	return fmt.Sprintf("[%s]", strings.Join(b, " "))
}

func writeNameArray(rs []Name) string {
	b := make([]string, len(rs))
	for i, a := range rs {
		b[i] = a.String()
	}
	return fmt.Sprintf("[%s]", strings.Join(b, " "))
}

func (pdf pdfWriter) dateString(t time.Time) string {
	_, tz := t.Zone()
	str := fmt.Sprintf("D:%d%02d%02d%02d%02d%02d+%02d'%02d'",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(),
		tz/60/60, tz/60%60)
	return pdf.encodeString(str, textString)
}

func (pdf pdfWriter) stringsArray(ar []string, mode stringEncoding) string {
	chunks := make([]string, len(ar))
	for i, val := range ar {
		chunks[i] = pdf.encodeString(val, mode)
	}
	return fmt.Sprintf("[%s]", strings.Join(chunks, " "))
}

// helper to shorten the writting of formatted strings
type buffer struct {
	*bytes.Buffer
}

func newBuffer() buffer {
	return buffer{Buffer: &bytes.Buffer{}}
}

func (b buffer) fmt(format string, arg ...interface{}) {
	fmt.Fprintf(b.Buffer, format, arg...)
}

// add a formatted line
func (b buffer) line(format string, arg ...interface{}) {
	b.fmt(format, arg...)
	b.WriteByte('\n')
}