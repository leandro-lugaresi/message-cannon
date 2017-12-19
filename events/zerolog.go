package events

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type ZeroLogHandler struct {
	log zerolog.Logger
}

func (h *ZeroLogHandler) Handle(msg Message) {
	event := h.log.WithLevel(zerolog.Level(msg.Level))
	for _, v := range msg.Fields {
		switch val := v.value.(type) {
		case string:
			event.Str(v.key, val)
		case []byte:
			event.Bytes(v.key, val)
		case error:
			event.Err(val)
		case bool:
			event.Bool(v.key, val)
		case int:
			event.Int(v.key, val)
		case int32:
			event.Int32(v.key, val)
		case int64:
			event.Int64(v.key, val)
		case float32:
			event.Float32(v.key, val)
		case float64:
			event.Float64(v.key, val)
		case time.Time:
			event.Time(v.key, val)
		case time.Duration:
			event.Dur(v.key, val)
		case nil:
		default:
			event.Interface(v.key, val)
		}
	}
	event.Msg(msg.Msg)
}

func (h *ZeroLogHandler) Sync() {}

func NewDefaultLogger(development bool) Handler {
	var w io.Writer
	w = os.Stdout
	if development {
		w = zerolog.ConsoleWriter{Out: os.Stdout}
	}
	return &ZeroLogHandler{
		log: zerolog.New(w).With().
			Timestamp().
			Str("app", "message-cannon").
			Logger(),
	}
}
