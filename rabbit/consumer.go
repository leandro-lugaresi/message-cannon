package rabbit

import (
	"context"
	"io"

	"gopkg.in/tomb.v2"

	"github.com/streadway/amqp"
)

type consumer struct {
	runner      func(context.Context, io.Writer, []byte) error
	hash        string
	name        string
	queue       string
	factoryName string
	opts        Options
	channel     *amqp.Channel
	t           tomb.Tomb
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
			return err
		}
		for {
			select {
			case <-c.t.Dying():
				return nil
			case err := <-c.channel.NotifyClose(make(chan *amqp.Error)):
				return err
			case <-d:

				//TODO: Process the message :p
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
