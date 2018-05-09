package subscriber

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/leandro-lugaresi/hub"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestLogSubscriber(t *testing.T) {
	tests := []struct {
		name   string
		msg    hub.Message
		result string
	}{
		{
			"field with string",
			hub.Message{
				Name:   "foo.baz.bar",
				Body:   []byte("info message"),
				Fields: hub.Fields{"foo": "baz"},
			},
			`{"level":"debug","foo":"baz","message":"info message"}`,
		},
		{
			"field with floats",
			hub.Message{
				Name:   "foo.baz.warning",
				Body:   []byte("fooooo"),
				Fields: hub.Fields{"foo": float64(123.43), "other": float32(222.34)},
			},
			`{"level":"warn","foo":123.43,"other":222.34,"message":"fooooo"}`,
		},
		{
			"field with integers",
			hub.Message{
				Name:   "foo.baz.error",
				Body:   []byte("fooooo"),
				Fields: hub.Fields{"a": int64(123), "b": int32(222), "c": 0},
			},
			`{"level":"error","b":222,"a":123,"c":0,"message":"fooooo"}`,
		},
		{
			"field with bytes",
			hub.Message{
				Name:   "foo.baz.info",
				Body:   []byte("fooooo"),
				Fields: hub.Fields{"foo": []byte(`<h1>something</h1>`)},
			},
			`{"level":"info","foo":"<h1>something</h1>","message":"fooooo"}`,
		},
		{
			"field with error and boolean",
			hub.Message{
				Name:   "foo.baz.info",
				Body:   []byte("fooooo"),
				Fields: hub.Fields{"error": errors.New(`something failed`), "foo": true},
			},
			`{"level":"info","error":"something failed","foo":true,"message":"fooooo"}`,
		},
		{
			"fields with Time and Duration",
			hub.Message{
				Name:   "foo.baz.info",
				Body:   []byte("fooooo"),
				Fields: hub.Fields{"time": time.Date(2018, 1, 18, 2, 27, 0, 0, time.UTC), "duration": time.Second},
			},
			`{"level":"info","time":"2018-01-18T02:27:00Z","duration":1000,"message":"fooooo"}`,
		},
		{
			"field with some struct",
			hub.Message{
				Name:   "foo.baz.success",
				Body:   []byte("fooooo"),
				Fields: hub.Fields{"struct": struct{ Key, Value string }{Key: "fooo", Value: "baz"}},
			},
			`{"level":"debug","struct": {"Key":"fooo","Value":"baz"},"message":"fooooo"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := hub.New()
			w := &bytes.Buffer{}
			log := &Logger{
				Logger: zerolog.New(w),
				sub:    h.Subscribe(0, "*.*.*"),
				done:   make(chan struct{}),
			}
			go log.Do()
			h.Publish(tt.msg)
			h.Close()
			log.Stop()
			require.JSONEq(t, tt.result, w.String())
		})
	}
}
