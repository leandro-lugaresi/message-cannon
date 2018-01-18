package event

import (
	"io"
	"time"

	"github.com/rs/zerolog"
)

type ZeroLogHandler struct {
	log zerolog.Logger
}

func (h *ZeroLogHandler) Handle(msg Message) {
	event := h.log.WithLevel(zerolog.Level(msg.Level))
	for _, v := range msg.Fields {
		switch val := v.Value.(type) {
		case string:
			event.Str(v.Key, val)
		case []byte:
			event.Bytes(v.Key, val)
		case error:
			event.Err(val)
		case bool:
			event.Bool(v.Key, val)
		case int:
			event.Int(v.Key, val)
		case int32:
			event.Int32(v.Key, val)
		case int64:
			event.Int64(v.Key, val)
		case float32:
			event.Float32(v.Key, val)
		case float64:
			event.Float64(v.Key, val)
		case time.Time:
			event.Time(v.Key, val)
		case time.Duration:
			event.Dur(v.Key, val)
		case nil:
		default:
			event.Interface(v.Key, val)
		}
	}
	event.Msg(msg.Msg)
}

func (h *ZeroLogHandler) Sync() {}

// NewZeroLogHandler set the ZeroLog logger with sensible defaults as the default handler.
func NewZeroLogHandler(w io.Writer, development bool) Handler {
	if development {
		w = zerolog.ConsoleWriter{Out: w}
	}
	return &ZeroLogHandler{
		log: zerolog.New(w).With().
			Timestamp().
			Logger(),
	}
}
