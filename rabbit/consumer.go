package rabbit

import (
	"context"

	"gopkg.in/tomb.v2"

	"github.com/leandro-lugaresi/message-cannon/runner"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

type consumer struct {
	runner      runner.Runnable
	hash        string
	name        string
	queue       string
	factoryName string
	opts        Options
	channel     *amqp.Channel
	t           tomb.Tomb
	l           *zap.Logger
}

// Run will get the messages and pass to the runner.
func (c *consumer) Run() {
	c.t.Go(func() error {
		defer func() {
			err := c.channel.Close()
			if err != nil {
				c.l.Error("Error closing the consumer channel", zap.Error(err))
			}
		}()
		d, err := c.channel.Consume(c.queue, "rabbitmq-"+c.name+"-"+c.hash,
			c.opts.AutoAck,
			c.opts.Exclusive,
			c.opts.NoLocal,
			c.opts.NoWait,
			c.opts.Args)
		if err != nil {
			c.l.Error("Failed to start consume", zap.Error(err))
			return err
		}
		dying := c.t.Dying()
		closed := c.channel.NotifyClose(make(chan *amqp.Error))
		for {
			select {
			case <-dying:
				return nil
			case err := <-closed:
				return err
			case msg := <-d:
				c.processMessage(msg)
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

func (c *consumer) processMessage(msg amqp.Delivery) {
	status := c.runner.Process(context.Background(), msg.Body)
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
		c.l.Warn("The runner return an unexpected exitStatus and the message will be rejected.", zap.Int("status", status))
		err = msg.Reject(false)
	}
	if err != nil {
		c.l.Error("Error during the acknowledgement phase", zap.Error(err))
	}
}
