package event

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLogger_LogShouldSendAllLogsToHandler(t *testing.T) {
	messsages := []Message{}
	log := NewLogger(HandlerFunc(func(msg Message) {
		messsages = append(messsages, msg)
	}), 10)
	for i := 0; i < 10; i++ {
		log.Log(WarnLevel, "test message", Field{"i", i}, Field{"priest", "wololo"})
	}
	log.Close()
	for i, msg := range messsages {
		require.Equal(t, Message{
			Msg:   "test message",
			Level: WarnLevel,
			Fields: []Field{
				Field{"i", i},
				Field{"priest", "wololo"},
			},
		}, msg, "Wrong message received")
	}
}

func TestLogger_WithShouldCreateAnSubLoggerWithAllTheFields(t *testing.T) {
	messsages := []Message{}
	log := NewLogger(HandlerFunc(func(msg Message) {
		messsages = append(messsages, msg)
	}), 10)
	sub1 := log.With(Field{"foo", "baz"}, Field{"proc", "sub1"})
	sub2 := log.With(Field{"foo", "baz"}, Field{"proc", "sub2"})
	sub1.Log(InfoLevel, "test log sub1")
	sub2.Log(InfoLevel, "test log sub2")
	sub21 := sub2.With(Field{"inner-log", true})
	sub21.Log(InfoLevel, "test log sub21", Field{"id", 123})
	log.Close()
	require.Equal(t, 3, len(messsages))
	require.Equal(t, Message{
		Msg:   "test log sub1",
		Level: InfoLevel,
		Fields: []Field{
			Field{"foo", "baz"},
			Field{"proc", "sub1"},
		},
	}, messsages[0], "Receive invalid message for sub1")
	require.Equal(t, Message{
		Msg:   "test log sub2",
		Level: InfoLevel,
		Fields: []Field{
			Field{"foo", "baz"},
			Field{"proc", "sub2"},
		},
	}, messsages[1], "Receive invalid message for sub2")
	require.Equal(t, Message{
		Msg:   "test log sub21",
		Level: InfoLevel,
		Fields: []Field{
			Field{"id", 123},
			Field{"foo", "baz"},
			Field{"proc", "sub2"},
			Field{"inner-log", true},
		},
	}, messsages[2], "Receive invalid message for sub21")
}

func BenchmarkLogFirstLevel(b *testing.B) {
	log := NewLogger(NewNoOpHandler(), b.N)
	for n := 0; n < b.N; n++ {
		log.Log(InfoLevel, "message log for benchmark", Field{"n", n}, Field{"bench1", true})
	}
	log.Close()
}

func BenchmarkLogZeroLog(b *testing.B) {
	log := NewLogger(NewZeroLogHandler(ioutil.Discard, false), b.N)
	for n := 0; n < b.N; n++ {
		log.Log(InfoLevel, "message log for benchmark", Field{"n", n}, Field{"bench1", true})
	}
	log.Close()
}
