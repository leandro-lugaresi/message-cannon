package worker

import (
	"fmt"
	"strings"

	"gopkg.in/tomb.v2"
)

type stubFactory struct {
	name             string
	connectionClosed chan error
	qtdConsumers     int
}

type stubConsumer struct {
	t                tomb.Tomb
	connectionClosed chan error
	name             string
	factoryName      string
	runCalled        bool
}

func newStubFactory(name string, qtdConsumers int) *stubFactory {
	return &stubFactory{
		name,
		make(chan error),
		qtdConsumers,
	}
}

func (f *stubFactory) CreateConsumers() ([]Consumer, error) {
	c := make([]Consumer, f.qtdConsumers)
	for i := 0; i < f.qtdConsumers; i++ {
		c[i] = newStubConsumer(fmt.Sprint(f.name, "-consumer-", i), f.name, f.connectionClosed)
	}
	return c, nil
}

func (f *stubFactory) CreateConsumer(name string) (Consumer, error) {
	if !strings.HasPrefix(name, f.name+"-consumer-") {
		return nil, fmt.Errorf("Consumer name not expected, expected: \"%s-consumer-\", received: \"%s\" ", f.name, name)
	}
	return newStubConsumer(name, f.name, f.connectionClosed), nil
}

func (f *stubFactory) Name() string {
	return f.name
}

// emulate a connection close
func (f *stubFactory) Close() {
	close(f.connectionClosed)
}

// emulate a connection reset
func (f *stubFactory) Reconnect() {
	f.connectionClosed = make(chan error)
}

func newStubConsumer(name, factoryName string, errChan chan error) *stubConsumer {
	return &stubConsumer{
		tomb.Tomb{},
		errChan,
		name,
		factoryName,
		false,
	}
}

func (c *stubConsumer) Run() {
	c.runCalled = true
	c.t.Go(func() error {
		for {
			select {
			case err := <-c.connectionClosed:
				return err
			case <-c.t.Dying():
				return nil
			}
		}
	})
}

func (c *stubConsumer) Kill() error {
	var err error
	c.t.Kill(err)
	<-c.t.Dead()
	return err
}

func (c *stubConsumer) Alive() bool {
	return c.t.Alive()
}

func (c *stubConsumer) Name() string {
	return c.name
}

func (c *stubConsumer) FactoryName() string {
	return c.factoryName
}
