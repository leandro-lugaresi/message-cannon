package rabbit

import (
	"gopkg.in/tomb.v2"

	"github.com/streadway/amqp"
)

type consumer struct {
	name        string
	factoryName string
	channel     *amqp.Channel
	t           tomb.Tomb
}

// Run will get the messages and pass to the runner.
func (c *consumer) Run() {

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
