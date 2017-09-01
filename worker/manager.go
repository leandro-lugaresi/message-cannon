package cmd

import (
	"context"
	"io"
)

// Runner encapsulates what is done for every message received.
type Runner func(context.Context, io.Writer, []byte) error

// Factory create consumers
type Factory interface {
	// CreateConsumers will interate over config and create all the consumers
	CreateConsumers() (Consumer, error)

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
	Run(Runner) error

	// Kill will try to stop the internal work. Return an error in case of failure.
	Kill() error

	// Name return the name for this consumer
	Name() string
}

// Manager is the block responsible for create all the consumers.
// Keeping track of the current state os consumers and stop/restart consumers when nedded.
type Manager struct {
	factories map[string]Factory
	consumers map[string]Consumer
}

// NewManager will setup all the consumers from fatories.
func NewManager(factories []Factory) (*Manager, error) {
	m := &Manager{}
	for _, f := range factories {
		m.factories[f.Name()] = f
		c, err := f.CreateConsumers()
		if err != nil {
			return nil, err
		}

	}
	return m, nil
}

// Start will initialize the consumers.
func (m *Manager) Start() error {

}

// Stop all the consumers
func (m *Manager) Stop() error {

	return nil
}
