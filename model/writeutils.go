package model

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// FmtFloat returns a PDF compatible float representation of `f`.
func FmtFloat(f Fl) string {
	// avoid to represent 0 as -0
	if f == 0 {
		return "0"
	}
	// Round rounds f with 12 digits precision
	n := math.Pow10(5)
	f_ := math.Round(float64(f)*n) / n

	return strconv.FormatFloat(f_, 'f', -1, 32)
}

func writeMaybeFloat(f MaybeFloat) string {
	if f == nil {
		return "null"
	}
	return FmtFloat(Fl(f.(ObjFloat)))
}

func writeIntArray(as []int) string {
	b := make([]string, len(as))
	for i, a := range as {
		b[i] = strconv.Itoa(a)
	}
	return fmt.Sprintf("[%s]", strings.Join(b, " "))
}

func writeFloatArray(as []Fl) string {
	b := make([]string, len(as))
	for i, a := range as {
		b[i] = FmtFloat(a)
	}
	return fmt.Sprintf("[%s]", strings.Join(b, " "))
}

func writeRefArray(as []Reference) string {
	b := make([]string, len(as))
	for i, ref := range as {
		b[i] = ref.String()
	}
	return fmt.Sprintf("[%s]", strings.Join(b, " "))
}

func writePointArray(rs [][2]Fl) string {
	b := make([]string, len(rs))
	for i, a := range rs {
		b[i] = fmt.Sprintf("%s %s ", FmtFloat(a[0]), FmtFloat(a[1]))
	}
	return fmt.Sprintf("[%s]", strings.Join(b, " "))
}

func writeRangeArray(rs []Range) string {
	b := make([]string, len(rs))
	for i, a := range rs {
		b[i] = fmt.Sprintf("%s %s", FmtFloat(a[0]), FmtFloat(a[1]))
	}
	return fmt.Sprintf("[%s]", strings.Join(b, " "))
}

func writePointsArray(rs [][2]Fl) string {
	b := make([]string, len(rs))
	for i, a := range rs {
		b[i] = fmt.Sprintf("%s %s ", FmtFloat(a[0]), FmtFloat(a[1]))
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

// DateTimeString returns a valid PDF string representation of `t`.
// Note that the string is not encoded (or crypted).
func DateTimeString(t time.Time) string {
	_, tz := t.Zone()
	tzm := tz / 60
	sign := "+"
	if tzm < 0 {
		sign = "-"
		tzm = -tzm
	}
	return fmt.Sprintf("D:%d%02d%02d%02d%02d%02d%s%02d'%02d'",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(),
		sign, tzm/60, tzm%60)
}

func (pdf pdfWriter) dateString(t time.Time, context Reference) string {
	return pdf.EncodeString(DateTimeString(t), TextString, context)
}

func writeStringsArray(ar []string, pdf PDFWritter, mode PDFStringEncoding, context Reference) string {
	chunks := make([]string, len(ar))
	for i, val := range ar {
		chunks[i] = pdf.EncodeString(val, mode, context)
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
