package rabbit

import (
	"testing"

	"github.com/streadway/amqp"
	"github.com/stretchr/testify/require"
)

func Test_getHeaders(t *testing.T) {
	tests := []struct {
		name string
		args amqp.Delivery
		want map[string]string
	}{
		{
			"with empty headers",
			amqp.Delivery{Body: []byte(`foooo`)},
			map[string]string{
				"Content-Encoding": "",
				"Content-Type":     "",
				"Correlation-Id":   "",
				"Message-Id":       "",
			},
		},
		{
			"with headers",
			amqp.Delivery{
				ContentEncoding: "compress, gzip",
				ContentType:     "application/json",
				CorrelationId:   "id-12334455",
				MessageId:       "12345566",
				Body:            []byte(`foooo`),
			},
			map[string]string{
				"Content-Encoding": "compress, gzip",
				"Content-Type":     "application/json",
				"Correlation-Id":   "id-12334455",
				"Message-Id":       "12345566",
			},
		},
		{
			"with message headers x-death empty",
			amqp.Delivery{
				Body: []byte(`foooo`),
				Headers: amqp.Table{
					"x-death": []amqp.Table{},
				},
			},
			map[string]string{
				"Content-Encoding":    "",
				"Content-Type":        "",
				"Correlation-Id":      "",
				"Message-Id":          "",
				"Message-Death-Count": "0",
			},
		},
		{
			"with message headers x-death empty",
			amqp.Delivery{
				Body: []byte(`foooo`),
				Headers: amqp.Table{
					"x-death": []amqp.Table{
						amqp.Table{"time": "2018-02-13T17:50:26-02:00", "count": 4, "exchange": "fallback", "queue": "fallback", "reason": "expired"},
						amqp.Table{"time": "2018-02-13T17:50:34-02:00", "count": 1, "exchange": "fallback", "queue": "fallback", "reason": "rejected"},
						amqp.Table{"time": "2018-02-13T17:45:26-02:00", "count": 5, "exchange": "fallback", "queue": "GenerateReport", "reason": "rejected"},
					},
				},
			},
			map[string]string{
				"Content-Encoding":    "",
				"Content-Type":        "",
				"Correlation-Id":      "",
				"Message-Id":          "",
				"Message-Death-Count": "6",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getHeaders(tt.args)
			require.Exactly(t, tt.want, got)
		})
	}
}
