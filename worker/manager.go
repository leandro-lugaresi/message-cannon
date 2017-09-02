package cmd

import (
	"context"
	"io"
)

// Runner encapsulate what is done with messages
type Runner func(context.Context, io.Writer, []byte) error

// Factory create consumers
type Factory interface {
	// CreateConsumers will iterate over config and create all the consumers
	CreateConsumers() ([]Consumer, error)

	// CreateConsumer create a new consumer for a specific name using the config provided.
	CreateConsumer(name string) (Consumer, error)

	// Name return the factory name
	Name() string
}

// Consumer consume messages and pass to workers who will process the messages.
type Consumer interface {
	// TODO: Create the state, we will add some metrics here
	// State returns a copy of the executor's current operation state.
	// State() State

	// Run will get the messages and pass to the runner.
	Run() error

	// Kill will try to stop the internal work. Return an error in case of failure.
	Kill() error

	// Name return the consumer name
	Name() string
}

// Manager is the block responsible for creating all the consumers.
// Keeping track of the current state of consumers and stop/restart consumers when needed.
type Manager struct {
	ops chan func(map[string]Factory, map[string]Consumer)
}

// NewManager will init a new manager and wait for operations.
func NewManager() (*Manager, error) {
	m := &Manager{}
	go m.work()
	return m, nil
}

func (m *Manager) work() {
	factories := make(map[string]Factory)
	consumers := make(map[string]Consumer)
	for op := range m.ops {
		op(factories, consumers)
	}
}

// Start will all the consumers from factories
func (m *Manager) Start(fs []Factory) error {
	var err error
	m.ops <- func(factories map[string]Factory, consumers map[string]Consumer) {
		for _, f := range fs {
			factories[f.Name()] = f
			cs, err := f.CreateConsumers()
			if err != nil {
				return
			}
			for _, c := range cs {
				consumers[c.Name()] = c
			}
		}
	}
	return err
}

// Stop all the consumers
func (m *Manager) Stop() error {
	var err error
	m.ops <- func(factories map[string]Factory, consumers map[string]Consumer) {
		for _, c := range consumers {
			err = c.Kill()
		}
	}
	return err
}
