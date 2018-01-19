package rabbit

import (
	"context"
	"errors"
	"sync"
	"time"

	"gopkg.in/tomb.v2"

	"github.com/leandro-lugaresi/message-cannon/event"
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
	l           *event.Logger
}

// Run will get the messages and pass to the runner.
func (c *consumer) Run() {
	c.t.Go(func() error {
		defer func() {
			err := c.channel.Close()
			if err != nil {
				c.l.Error("Error closing the consumer channel", event.KV("error", err))
			}
		}()
		d, err := c.channel.Consume(c.queue, "rabbitmq-"+c.name+"-"+c.hash,
			c.opts.AutoAck,
			c.opts.Exclusive,
			c.opts.NoLocal,
			c.opts.NoWait,
			c.opts.Args)
		if err != nil {
			c.l.Error("Failed to start consume", event.KV("error", err))
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
	status := c.runner.Process(ctx, msg.Body)
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
		c.l.Warn("The runner return an unexpected exitStatus and the message will be requeued.", event.KV("status", status))
		err = msg.Reject(true)
	}
	if err != nil {
		c.l.Error("Error during the acknowledgement phase", event.KV("error", err))
	}
}
