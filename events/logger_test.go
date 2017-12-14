package events

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLogger_LogShouldSendAllLogsToHandler(t *testing.T) {
	messsages := []Message{}
	log := NewLogger(func(msg Message) {
		messsages = append(messsages, msg)
	}, 10)
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
