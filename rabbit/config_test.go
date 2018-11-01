package rabbit

import (
	"testing"
	"time"

	"github.com/leandro-lugaresi/message-cannon/runner"
	"github.com/stretchr/testify/require"
)

func Test_withDefaults(t *testing.T) {
	config := Config{
		Connections: map[string]Connection{
			"de": {DSN: "amqp://localhost:5672"},
		},
		Consumers: map[string]ConsumerConfig{
			"consumer1": {
				Connection: "server1",
				Queue:      QueueConfig{Name: "fooo"},
				Runner: runner.Config{
					Type: "http",
					Options: runner.Options{
						URL: "http://localhost:8080",
					},
				},
			},
		},
		Version: "0.0.5",
	}

	err := setConfigDefaults(&config)
	require.NoError(t, err)
	require.Equal(t, 5, config.Connections["de"].Retries)
	require.Equal(t, 2*time.Second, config.Connections["de"].Timeout)
	require.Equal(t, 500*time.Millisecond, config.Connections["de"].Sleep)
	require.Equal(t, 1, config.Consumers["consumer1"].MaxWorkers)
	require.Equal(t, 10, config.Consumers["consumer1"].PrefetchCount)
	require.Equal(t, 4, config.Consumers["consumer1"].Runner.Options.ReturnOn5xx)
	require.Equal(t, "message-cannon/0.0.5", config.Consumers["consumer1"].Runner.Options.Headers["User-Agent"])
	config.Consumers["consumer1"].Runner.Options.Headers["User-Agent"] = "UserAgent From Config"
	err = setConfigDefaults(&config)
	require.NoError(t, err)
	require.Equal(t, "UserAgent From Config", config.Consumers["consumer1"].Runner.Options.Headers["User-Agent"])
}
