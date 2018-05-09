package subscriber

import (
	"io"
	"strings"
	"time"

	"github.com/leandro-lugaresi/hub"
	"github.com/rs/zerolog"
)

// Logger is wrapper around the zerolog.Logger with
// the ability to handle messages from hub.Subscription
type Logger struct {
	zerolog.Logger
	sub  hub.Subscription
	done chan struct{}
}

// Do will start consuming messages from the subscriber and stop when the Subscription is closed
func (l *Logger) Do() {
	for msg := range l.sub.Receiver {
		event := l.WithLevel(getLevel(msg.Name))
		for k, v := range msg.Fields {
			switch val := v.(type) {
			case string:
				event.Str(k, val)
			case []byte:
				event.Bytes(k, val)
			case error:
				event.Err(val)
			case bool:
				event.Bool(k, val)
			case int:
				event.Int(k, val)
			case int32:
				event.Int32(k, val)
			case int64:
				event.Int64(k, val)
			case float32:
				event.Float32(k, val)
			case float64:
				event.Float64(k, val)
			case time.Time:
				event.Time(k, val)
			case time.Duration:
				event.Dur(k, val)
			case nil:
			default:
				event.Interface(k, val)
			}
		}
		event.Msg(string(msg.Body))
	}
	close(l.done)
}

// Stop close any open file and clean stuffs
func (l *Logger) Stop() {
	<-l.done
}

func getLevel(topic string) zerolog.Level {
	switch {
	case strings.HasSuffix(topic, ".info"):
		return zerolog.InfoLevel
	case strings.HasSuffix(topic, ".error"):
		return zerolog.ErrorLevel
	case strings.HasSuffix(topic, ".warning"):
		return zerolog.WarnLevel
	}
	return zerolog.DebugLevel
}

// NewLogger create an Logger.
func NewLogger(w io.Writer, sub hub.Subscription, development bool) *Logger {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if development {
		w = zerolog.ConsoleWriter{Out: w}
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	return &Logger{
		Logger: zerolog.New(w).With().Timestamp().Logger(),
		sub:    sub,
		done:   make(chan struct{}),
	}
}
