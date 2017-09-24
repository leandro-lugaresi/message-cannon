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
		defer c.channel.Close()
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
		for {
			select {
			case <-c.t.Dying():
				return nil
			case err := <-c.channel.NotifyClose(make(chan *amqp.Error)):
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
	switch status {
	case runner.ExitACK:
		msg.Ack(false)
	case runner.ExitFailed:
		msg.Reject(true)
	case runner.ExitRetry, runner.ExitNACKRequeue, runner.ExitTimeout:
		msg.Nack(false, true)
	case runner.ExitNACK:
		msg.Nack(false, false)
	default:
		c.l.Warn("The runner return an unexpected exitStatus and the message will be rejected.", zap.Int("status", status))
		msg.Reject(false)
	}
}
