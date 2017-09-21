package diagnostic_test

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/influxdata/kapacitor/services/diagnostic"
)

var defaultTime = time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)

type testStringer string

func (t testStringer) String() string {
	return string(t)
}

func TestLoggerWithoutContext(t *testing.T) {
	now := time.Now()
	nowStr := now.Format(diagnostic.RFC3339Milli)
	buf := bytes.NewBuffer(nil)
	l := diagnostic.NewServerLogger(buf)

	tests := []struct {
		name   string
		exp    string
		lvl    string
		msg    string
		fields []diagnostic.Field
	}{
		{
			name: "no fields simple message",
			exp:  fmt.Sprintf("ts=%s lvl=error msg=this\n", nowStr),
			lvl:  "error",
			msg:  "this",
		},
		{
			name: "no fields less simple message",
			exp:  fmt.Sprintf("ts=%s lvl=error msg=this/is/a/test\n", nowStr),
			lvl:  "error",
			msg:  "this/is/a/test",
		},
		{
			name: "no fields complex message",
			exp:  fmt.Sprintf("ts=%s lvl=error msg=\"this is \\\" a test/yeah\"\n", nowStr),
			lvl:  "error",
			msg:  "this is \" a test/yeah",
		},
		{
			name: "simple string field",
			exp:  fmt.Sprintf("ts=%s lvl=error msg=test test=this\n", nowStr),
			lvl:  "error",
			msg:  "test",
			fields: []diagnostic.Field{
				diagnostic.String("test", "this"),
			},
		},
		{
			name: "complex string field",
			exp:  fmt.Sprintf("ts=%s lvl=error msg=test test=\"this is \\\" a test/yeah\"\n", nowStr),
			lvl:  "error",
			msg:  "test",
			fields: []diagnostic.Field{
				diagnostic.String("test", "this is \" a test/yeah"),
			},
		},
		{
			name: "simple stringer field",
			exp:  fmt.Sprintf("ts=%s lvl=error msg=test test=this\n", nowStr),
			lvl:  "error",
			msg:  "test",
			fields: []diagnostic.Field{
				diagnostic.Stringer("test", testStringer("this")),
			},
		},
		{
			name: "simple single grouped field",
			exp:  fmt.Sprintf("ts=%s lvl=error msg=test test_a=this\n", nowStr),
			lvl:  "error",
			msg:  "test",
			fields: []diagnostic.Field{
				diagnostic.GroupedFields("test", []diagnostic.Field{
					diagnostic.String("a", "this"),
				}),
			},
		},
		{
			name: "simple double grouped field",
			exp:  fmt.Sprintf("ts=%s lvl=error msg=test test_a=this test_b=other\n", nowStr),
			lvl:  "error",
			msg:  "test",
			fields: []diagnostic.Field{
				diagnostic.GroupedFields("test", []diagnostic.Field{
					diagnostic.String("a", "this"),
					diagnostic.String("b", "other"),
				}),
			},
		},
		{
			name: "simple single strings field",
			exp:  fmt.Sprintf("ts=%s lvl=error msg=test test_0=this\n", nowStr),
			lvl:  "error",
			msg:  "test",
			fields: []diagnostic.Field{
				diagnostic.Strings("test", []string{"this"}),
			},
		},
		{
			name: "simple double strings field",
			exp:  fmt.Sprintf("ts=%s lvl=error msg=test test_0=this test_1=other\n", nowStr),
			lvl:  "error",
			msg:  "test",
			fields: []diagnostic.Field{
				diagnostic.Strings("test", []string{"this", "other"}),
			},
		},
		{
			name: "int field",
			exp:  fmt.Sprintf("ts=%s lvl=error msg=test test=10\n", nowStr),
			lvl:  "error",
			msg:  "test",
			fields: []diagnostic.Field{
				diagnostic.Int("test", 10),
			},
		},
		{
			name: "int64 field",
			exp:  fmt.Sprintf("ts=%s lvl=error msg=test test=10\n", nowStr),
			lvl:  "error",
			msg:  "test",
			fields: []diagnostic.Field{
				diagnostic.Int64("test", 10),
			},
		},
		{
			name: "float64 field",
			exp:  fmt.Sprintf("ts=%s lvl=error msg=test test=3.1415926535\n", nowStr),
			lvl:  "error",
			msg:  "test",
			fields: []diagnostic.Field{
				diagnostic.Float64("test", 3.1415926535),
			},
		},
		{
			name: "bool true field",
			exp:  fmt.Sprintf("ts=%s lvl=error msg=test test=true\n", nowStr),
			lvl:  "error",
			msg:  "test",
			fields: []diagnostic.Field{
				diagnostic.Bool("test", true),
			},
		},
		{
			name: "bool false field",
			exp:  fmt.Sprintf("ts=%s lvl=error msg=test test=false\n", nowStr),
			lvl:  "error",
			msg:  "test",
			fields: []diagnostic.Field{
				diagnostic.Bool("test", false),
			},
		},
		{
			name: "simple error field",
			exp:  fmt.Sprintf("ts=%s lvl=error msg=test err=this\n", nowStr),
			lvl:  "error",
			msg:  "test",
			fields: []diagnostic.Field{
				diagnostic.Error(errors.New("this")),
			},
		},
		{
			name: "nil error field",
			exp:  fmt.Sprintf("ts=%s lvl=error msg=test err=nil\n", nowStr),
			lvl:  "error",
			msg:  "test",
			fields: []diagnostic.Field{
				diagnostic.Error(nil),
			},
		},
		{
			name: "complex error field",
			exp:  fmt.Sprintf("ts=%s lvl=error msg=test err=\"this is \\\" a test/yeah\"\n", nowStr),
			lvl:  "error",
			msg:  "test",
			fields: []diagnostic.Field{
				diagnostic.Error(errors.New("this is \" a test/yeah")),
			},
		},
		{
			name: "time field",
			exp:  fmt.Sprintf("ts=%s lvl=error msg=test time=%s\n", nowStr, defaultTime.Format(time.RFC3339Nano)),
			lvl:  "error",
			msg:  "test",
			fields: []diagnostic.Field{
				diagnostic.Time("time", defaultTime),
			},
		},
		{
			name: "duration field",
			exp:  fmt.Sprintf("ts=%s lvl=error msg=test test=1s\n", nowStr),
			lvl:  "error",
			msg:  "test",
			fields: []diagnostic.Field{
				diagnostic.Duration("test", time.Second),
			},
		},
		{
			name: "two fields",
			exp:  fmt.Sprintf("ts=%s lvl=error msg=test testing=\"that this\" works=1s\n", nowStr),
			lvl:  "error",
			msg:  "test",
			fields: []diagnostic.Field{
				diagnostic.String("testing", "that this"),
				diagnostic.Duration("works", time.Second),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer buf.Reset()
			l.Log(now, test.lvl, test.msg, test.fields)
			if exp, got := test.exp, buf.String(); exp != got {
				t.Fatalf("bad log line:\nexp: `%v`\ngot: `%v`", strconv.Quote(exp), strconv.Quote(got))
			}
		})
	}
}

func TestLoggerWithContext(t *testing.T) {
	now := time.Now()
	nowStr := now.Format(diagnostic.RFC3339Milli)
	buf := bytes.NewBuffer(nil)
	l := diagnostic.NewServerLogger(buf).With(diagnostic.String("a", "tag"), diagnostic.Int("id", 10)).(*diagnostic.ServerLogger)

	tests := []struct {
		name   string
		exp    string
		lvl    string
		msg    string
		fields []diagnostic.Field
	}{
		{
			name: "no fields simple message",
			exp:  fmt.Sprintf("ts=%s lvl=error msg=this a=tag id=10\n", nowStr),
			lvl:  "error",
			msg:  "this",
		},
		{
			name: "simple double grouped field",
			exp:  fmt.Sprintf("ts=%s lvl=error msg=test a=tag id=10 test_a=this test_b=other\n", nowStr),
			lvl:  "error",
			msg:  "test",
			fields: []diagnostic.Field{
				diagnostic.GroupedFields("test", []diagnostic.Field{
					diagnostic.String("a", "this"),
					diagnostic.String("b", "other"),
				}),
			},
		},
		{
			name: "simple double strings field",
			exp:  fmt.Sprintf("ts=%s lvl=error msg=test a=tag id=10 test_0=this test_1=other\n", nowStr),
			lvl:  "error",
			msg:  "test",
			fields: []diagnostic.Field{
				diagnostic.Strings("test", []string{"this", "other"}),
			},
		},
		{
			name: "two fields",
			exp:  fmt.Sprintf("ts=%s lvl=error msg=test a=tag id=10 testing=\"that this\" works=1s\n", nowStr),
			lvl:  "error",
			msg:  "test",
			fields: []diagnostic.Field{
				diagnostic.String("testing", "that this"),
				diagnostic.Duration("works", time.Second),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer buf.Reset()
			l.Log(now, test.lvl, test.msg, test.fields)
			if exp, got := test.exp, buf.String(); exp != got {
				t.Fatalf("bad log line:\nexp: `%v`\ngot: `%v`", strconv.Quote(exp), strconv.Quote(got))
			}
		})
	}
}

// TODO: is there something better than this?
func TestLogger_SetLeveF(t *testing.T) {
	var logLine string
	buf := bytes.NewBuffer(nil)
	l := diagnostic.NewServerLogger(buf)
	msg := "the message"

	l.SetLevelF(func(lvl diagnostic.Level) bool {
		return lvl >= diagnostic.DebugLevel
	})
	l.Debug(msg)
	logLine = buf.String()
	buf.Reset()
	if logLine == "" {
		t.Fatal("expected debug log")
		return
	}
	l.Info(msg)
	logLine = buf.String()
	buf.Reset()
	if logLine == "" {
		t.Fatal("expected info log")
		return
	}
	buf.Reset()
	l.Warn(msg)
	logLine = buf.String()
	if logLine == "" {
		t.Fatal("expected warn log")
		return
	}
	l.Error(msg)
	logLine = buf.String()
	buf.Reset()
	if logLine == "" {
		t.Fatal("expected error log")
		return
	}

	l.SetLevelF(func(lvl diagnostic.Level) bool {
		return lvl >= diagnostic.InfoLevel
	})
	l.Debug(msg)
	logLine = buf.String()
	buf.Reset()
	if logLine != "" {
		t.Fatal("expected no debug log")
		return
	}
	l.Info(msg)
	logLine = buf.String()
	buf.Reset()
	if logLine == "" {
		t.Fatal("expected info log")
		return
	}
	buf.Reset()
	l.Warn(msg)
	logLine = buf.String()
	if logLine == "" {
		t.Fatal("expected warn log")
		return
	}
	l.Error(msg)
	logLine = buf.String()
	buf.Reset()
	if logLine == "" {
		t.Fatal("expected error log")
		return
	}

	l.SetLevelF(func(lvl diagnostic.Level) bool {
		return lvl >= diagnostic.WarnLevel
	})
	l.Debug(msg)
	logLine = buf.String()
	buf.Reset()
	if logLine != "" {
		t.Fatal("expected no debug log")
		return
	}
	l.Info(msg)
	logLine = buf.String()
	buf.Reset()
	if logLine != "" {
		t.Fatal("expected no info log")
		return
	}
	buf.Reset()
	l.Warn(msg)
	logLine = buf.String()
	if logLine == "" {
		t.Fatal("expected warn log")
		return
	}
	l.Error(msg)
	logLine = buf.String()
	buf.Reset()
	if logLine == "" {
		t.Fatal("expected error log")
		return
	}

	l.SetLevelF(func(lvl diagnostic.Level) bool {
		return lvl >= diagnostic.ErrorLevel
	})
	l.Debug(msg)
	logLine = buf.String()
	buf.Reset()
	if logLine != "" {
		t.Fatal("expected no debug log")
		return
	}
	l.Info(msg)
	logLine = buf.String()
	buf.Reset()
	if logLine != "" {
		t.Fatal("expected no info log")
		return
	}
	buf.Reset()
	l.Warn(msg)
	logLine = buf.String()
	if logLine != "" {
		t.Fatal("expected no warn log")
		return
	}
	l.Error(msg)
	logLine = buf.String()
	buf.Reset()
	if logLine == "" {
		t.Fatal("expected error log")
		return
	}
}
