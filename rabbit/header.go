package rabbit

import (
	"strconv"
	"time"

	"github.com/leandro-lugaresi/message-cannon/runner"
	"github.com/streadway/amqp"
)

func getHeaders(msg amqp.Delivery) runner.Headers {
	headers := runner.Headers{
		"Content-Type":     msg.ContentType,
		"Content-Encoding": msg.ContentEncoding,
		"Correlation-Id":   msg.CorrelationId,
		"Message-Id":       msg.MessageId,
	}
	for k, v := range msg.Headers {
		switch vt := v.(type) {
		case int, int16, int32, int64, float32, float64, string, []byte, time.Time, bool:
			headers[k] = vt
		}
	}
	xdeaths, ok := msg.Headers["x-death"].([]interface{})
	if ok {
		headers["Message-Deaths"] = processDeaths(xdeaths)
	}

	return headers
}

func processDeaths(xdeaths []interface{}) string {
	var (
		count, deathCount int64
	)
	for _, ideath := range xdeaths {
		xdeath, ok := ideath.(amqp.Table)
		if !ok {
			continue
		}
		if xdeath["reason"] != "expired" {
			count, _ = xdeath["count"].(int64)
			deathCount += count
		}
	}
	return strconv.FormatInt(deathCount, 10)
}
