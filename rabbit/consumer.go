package rabbit

import (
	"context"
	"errors"
	"strconv"
	"sync"
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
	throttle    chan struct{}
	timeout     time.Duration
	factoryName string
	opts        Options
	channel     *amqp.Channel
	t           tomb.Tomb
	hub         *hub.Hub
}

// Run will get the messages and pass to the runner.
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
		var wg sync.WaitGroup
		for {
			select {
			case <-dying:
				wg.Wait()
				return nil
			case err := <-closed:
				return err
			case msg := <-d:
				if msg.Acknowledger == nil {
					c.hub.Publish(hub.Message{
						Name: "rabbit.consumer.error",
						Body: []byte("receive an empty delivery. closing consumer"),
					})
					return errors.New("receive an empty delivery")
				}
				c.throttle <- struct{}{}
				wg.Add(1)
				go func(msg amqp.Delivery) {
					nctx := ctx
					if c.timeout >= time.Second {
						var canc context.CancelFunc
						nctx, canc = context.WithTimeout(ctx, c.timeout)
						defer canc()
					}
					c.processMessage(nctx, msg)
					<-c.throttle
					wg.Done()
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
	status := c.runner.Process(ctx, msg.Body, getHeaders(msg))
	var err error
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
