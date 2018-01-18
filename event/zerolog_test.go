package event

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestZeroLogHandler_Handle(t *testing.T) {
	tests := []struct {
		name   string
		fields []Field
		result string
	}{
		{"field with string",
			[]Field{KV("foo", "baz")}, `"level":"info","foo":"baz","message":"info message"}`},
		{"fields with floats",
			[]Field{KV("foo", float64(123.43)), KV("other", float32(222.34))},
			`"level":"info","foo":123.43,"other":222.34,"message":"info message"}`},
		{"fields with integers",
			[]Field{KV("int32", int32(123)), KV("int64", int64(666)), KV("int", 0)},
			`"level":"info","int32":123,"int64":666,"int":0,"message":"info message"}`},
		{"field with bytes",
			[]Field{KV("foo", []byte(`<h1>something</h1>`))},
			`"level":"info","foo":"<h1>something</h1>","message":"info message"}`},
		{"field with error and boolean",
			[]Field{KV("error", errors.New(`something failed`)), KV("foo", true)},
			`"level":"info","error":"something failed","foo":true,"message":"info message"}`},
		{"field with Time and Duration",
			[]Field{KV("time", time.Date(2018, 1, 18, 2, 27, 0, 0, time.Local)), KV("duration", time.Second)},
			`"level":"info","time":"2018-01-18T02:27:00-02:00","duration":1000,"message":"info message"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			h := NewZeroLogHandler(w, false)
			h.Handle(Message{Msg: "info message", Level: InfoLevel, Fields: tt.fields})
			require.Contains(t, w.String(), tt.result)
		})
	}
}
