package rabbit

import (
	"testing"

	"github.com/leandro-lugaresi/message-cannon/runner"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/require"
)

func Test_getHeaders(t *testing.T) {
	tests := []struct {
		name string
		args amqp.Delivery
		want runner.Headers
	}{
		{
			"with empty headers",
			amqp.Delivery{Body: []byte(`foooo`)},
			runner.Headers{
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
			runner.Headers{
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
			runner.Headers{
				"Content-Encoding": "",
				"Content-Type":     "",
				"Correlation-Id":   "",
				"Message-Id":       "",
			},
		},
		{
			"with message headers x-death",
			amqp.Delivery{
				Body: []byte(`foooo`),
				Headers: amqp.Table{
					"x-death": []interface{}{
						amqp.Table{
							"time":     "2018-02-13T17:50:26-02:00",
							"count":    int64(4),
							"exchange": "fallback",
							"queue":    "fallback",
							"reason":   "expired"},
						amqp.Table{
							"time":     "2018-02-13T17:50:34-02:00",
							"count":    int64(1),
							"exchange": "fallback",
							"queue":    "fallback",
							"reason":   "rejected"},
						amqp.Table{
							"time":     "2018-02-13T17:45:26-02:00",
							"count":    int64(5),
							"exchange": "fallback",
							"queue":    "GenerateReport",
							"reason":   "rejected"},
					},
				},
			},
			runner.Headers{
				"Content-Encoding": "",
				"Content-Type":     "",
				"Correlation-Id":   "",
				"Message-Id":       "",
				"Message-Deaths":   "6",
			},
		},
		{
			"with custom headers",
			amqp.Delivery{
				Body: []byte(`foooo`),
				Headers: amqp.Table{
					"Authorization":   "Basic YWxhZGRpbjpvcGVuc2VzYW1l",
					"X-Forwarded-For": "203.0.113.195, 70.41.3.18, 150.172.238.178",
				},
			},
			runner.Headers{
				"Content-Encoding": "",
				"Content-Type":     "",
				"Correlation-Id":   "",
				"Message-Id":       "",
				"Authorization":    "Basic YWxhZGRpbjpvcGVuc2VzYW1l",
				"X-Forwarded-For":  "203.0.113.195, 70.41.3.18, 150.172.238.178",
			},
		},
	}
	for _, tt := range tests {
		ctt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := getHeaders(ctt.args)
			require.Exactly(t, ctt.want, got)
		})
	}
}
