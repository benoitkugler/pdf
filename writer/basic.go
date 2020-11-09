package writer

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/benoitkugler/pdf/model"
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

func writeRefArray(as []ref) string {
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
	return fmt.Sprintf("[ %s]", strings.Join(b, " "))
}

func writeRangeArray(rs []model.Range) string {
	b := make([]string, len(rs))
	for i, a := range rs {
		b[i] = fmt.Sprintf("%.3f %.3f ", a[0], a[1])
	}
	return fmt.Sprintf("[ %s]", strings.Join(b, " "))
}

func dateString(t time.Time) string {
	_, tz := t.Zone()
	return fmt.Sprintf("D:%d%02d%02d%02d%02d%02d+%02d'%02d'",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(),
		tz/60/60, tz/60%60)
}
