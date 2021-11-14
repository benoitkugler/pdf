package reader

import (
	"testing"
	"time"

	"github.com/benoitkugler/pdf/model"
)

func doParseDateTimeRelaxedOK(s string, t *testing.T) {
	t.Helper()
	if _, ok := DateTime(s); !ok {
		t.Errorf("DateTime(%s) invalid => not ok!\n", s)
	}
}

func doParseDateTimeOK(s string, t *testing.T) {
	t.Helper()
	if _, ok := dateTime(s, false); !ok {
		t.Errorf("DateTime(%s) invalid => not ok!\n", s)
	}
}

func doParseDateTimeFail(s string, t *testing.T) {
	t.Helper()
	if time, ok := dateTime(s, false); ok {
		t.Errorf("DateTime(%s) valid => not ok! %s\n", s, time)
	}
}

func TestParseDateTime(t *testing.T) {
	s := "D:2017"
	doParseDateTimeOK(s, t)

	s = "D:201703"
	doParseDateTimeOK(s, t)

	s = "D:20170430"
	doParseDateTimeOK(s, t)

	s = "D:2017043015"
	doParseDateTimeOK(s, t)

	s = "D:201704301559"
	doParseDateTimeOK(s, t)

	s = "D:20170430155901Z"
	doParseDateTimeOK(s, t)

	s = "D:20170430155901"
	doParseDateTimeOK(s, t)

	s = "D:20170430155901+06'59'"
	doParseDateTimeOK(s, t)

	s = "D:20170430155901Z00"
	doParseDateTimeOK(s, t)

	s = "D:20170430155901Z00'00'"
	doParseDateTimeOK(s, t)

	s = "D:20210602180254-06"
	doParseDateTimeOK(s, t)

	s = "D:20170430155901+06'"
	doParseDateTimeFail(s, t)

	s = "D:20170430155901+06'59"
	doParseDateTimeOK(s, t)

	s = "D:20210515103719-02'00"
	doParseDateTimeOK(s, t)

	s = "D:20170430155901+66'A9'"
	doParseDateTimeFail(s, t)

	s = "D:20201222164228Z'"
	doParseDateTimeRelaxedOK(s, t)

	s = "20141117162446Z00'00'"
	doParseDateTimeRelaxedOK(s, t)
}

func TestWriteDateTime(t *testing.T) {
	now := model.DateTimeString(time.Now())
	doParseDateTimeOK(now, t)

	loc, _ := time.LoadLocation("Europe/Vienna")
	now = model.DateTimeString(time.Now().In(loc))
	doParseDateTimeOK(now, t)

	loc, _ = time.LoadLocation("Pacific/Honolulu")
	now = model.DateTimeString(time.Now().In(loc))
	doParseDateTimeOK(now, t)

	loc, _ = time.LoadLocation("Australia/Sydney")
	now = model.DateTimeString(time.Now().In(loc))
	doParseDateTimeOK(now, t)
}
