package rabbit

import (
	"context"
	"errors"
	"strconv"
	"time"

	"gopkg.in/tomb.v2"

	"github.com/leandro-lugaresi/hub"
	"github.com/leandro-lugaresi/message-cannon/runner"
	"github.com/streadway/amqp"
)

type consumer struct {
	runner      runner.Runnable
	hash        string
	name        string
	queue       string
	workerPool  pool
	timeout     time.Duration
	factoryName string
	opts        Options
	channel     *amqp.Channel
	t           tomb.Tomb
	hub         *hub.Hub
}

// Run start a goroutine to consume messages and pass to one runner.
func (c *consumer) Run() {
	c.t.Go(func() error {
		defer func() {
			err := c.channel.Close()
			if err != nil {
				c.hub.Publish(hub.Message{
					Name:   "rabbit.consumer.error",
					Body:   []byte("Error closing the consumer channel"),
					Fields: hub.Fields{"error": err},
				})
			}
		}()
		d, err := c.channel.Consume(c.queue, "rabbitmq-"+c.name+"-"+c.hash,
			c.opts.AutoAck,
			c.opts.Exclusive,
			c.opts.NoLocal,
			c.opts.NoWait,
			c.opts.Args)
		if err != nil {
			c.hub.Publish(hub.Message{
				Name:   "rabbit.consumer.error",
				Body:   []byte("Failed to start consume"),
				Fields: hub.Fields{"error": err},
			})
			return err
		}
		dying := c.t.Dying()
		closed := c.channel.NotifyClose(make(chan *amqp.Error))
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		for {
			select {
			case <-dying:
				// When dying we wait for any remaining worker to finish
				c.workerPool.Wait()
				return nil
			case err := <-closed:
				return err
			case msg, ok := <-d:
				if !ok {
					c.hub.Publish(hub.Message{
						Name:   "rabbit.consumer.error",
						Body:   []byte("receive an empty delivery. closing consumer"),
						Fields: hub.Fields{},
					})
					return errors.New("receive an empty delivery")
				}
				// When maxWorkers goroutines are in flight, Acquire blocks until one of the
				// workers finishes.
				c.workerPool.Acquire()
				go func(msg amqp.Delivery) {
					nctx := ctx
					if c.timeout >= time.Second {
						var canc context.CancelFunc
						nctx, canc = context.WithTimeout(ctx, c.timeout)
						defer canc()
					}
					c.processMessage(nctx, msg)
					c.workerPool.Release()
				}(msg)
			}
		}
	})
}

// Kill will try to stop the internal work. Return an error in case of failure.
func (c *consumer) Kill() error {
	var err error
	c.t.Kill(err)
	<-c.t.Dead()
	return err
}

// Alive returns true if the tomb is not in a dying or dead state.
func (c *consumer) Alive() bool {
	return c.t.Alive()
}

// Name return the consumer name
func (c *consumer) Name() string {
	return c.name
}

// FactoryName is the name of the factory responsible for this consumer.
func (c *consumer) FactoryName() string {
	return c.factoryName
}

func (c *consumer) processMessage(ctx context.Context, msg amqp.Delivery) {
	var err error
	start := time.Now()
	status, err := c.runner.Process(ctx, runner.Message{Body: msg.Body, Headers: getHeaders(msg)})
	duration := time.Since(start)
	fields := hub.Fields{
		"duration":    duration,
		"status-code": status,
	}
	topic := "rabbit.process.sucess"
	if err != nil {
		topic = "rabbit.process.error"
		switch e := err.(type) {
		case *runner.Error:
			fields["error"] = e.Err
			fields["exit-code"] = e.StatusCode
			fields["output"] = e.Output
		default:
			fields["error"] = e
		}
	}
	c.hub.Publish(hub.Message{
		Name:   topic,
		Fields: fields,
	})
	switch status {
	case runner.ExitACK:
		err = msg.Ack(false)
	case runner.ExitFailed:
		err = msg.Reject(true)
	case runner.ExitRetry, runner.ExitNACKRequeue, runner.ExitTimeout:
		err = msg.Nack(false, true)
	case runner.ExitNACK:
		err = msg.Nack(false, false)
	default:
		c.hub.Publish(hub.Message{
			Name:   "rabbit.consumer.error",
			Body:   []byte("the runner returned an unexpected exitStatus. Message will be requeued."),
			Fields: hub.Fields{"status": status},
		})
		err = msg.Reject(true)
	}
	if err != nil {
		c.hub.Publish(hub.Message{
			Name:   "rabbit.consumer.error",
			Body:   []byte("error during the acknowledgement phase"),
			Fields: hub.Fields{"error": err},
		})
	}
}

func getHeaders(msg amqp.Delivery) map[string]string {
	headers := map[string]string{
		"Content-Type":     msg.ContentType,
		"Content-Encoding": msg.ContentEncoding,
		"Correlation-Id":   msg.CorrelationId,
		"Message-Id":       msg.MessageId,
	}
	xdeaths, ok := msg.Headers["x-death"].([]interface{})
	if !ok {
		return headers
	}
	var (
		count, deathCount int64
		xdeath            amqp.Table
	)
	for _, ideath := range xdeaths {
		xdeath, ok = ideath.(amqp.Table)
		if !ok {
			continue
		}
		if xdeath["reason"] != "expired" {
			count, _ = xdeath["count"].(int64)
			deathCount += count
		}
	}
	headers["Message-Deaths"] = strconv.FormatInt(deathCount, 10)
	return headers
}
