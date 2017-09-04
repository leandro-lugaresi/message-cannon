package worker

import (
	"context"
	"io"
	"sync"
	"time"
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
	Run()

	// Kill will try to stop the internal work. Return an error in case of failure.
	Kill() error

	// Alive returns true if the tomb is not in a dying or dead state.
	Alive() bool

	// Name return the consumer name
	Name() string

	// FactoryName is the name of the factory responsible for this consumer.
	FactoryName() string
}

// Manager is the block responsible for creating all the consumers.
// Keeping track of the current state of consumers and stop/restart consumers when needed.
type Manager struct {
	checkAliveness time.Duration
	ops            chan func(map[string]Factory, map[string]Consumer)
}

// NewManager init a new manager and wait for operations.
func NewManager(intervalChecks time.Duration) *Manager {
	m := &Manager{
		checkAliveness: intervalChecks,
		ops:            make(chan func(map[string]Factory, map[string]Consumer)),
	}
	go m.work()
	go m.checkConsumers()
	return m
}

func (m *Manager) work() {
	factories := make(map[string]Factory)
	consumers := make(map[string]Consumer)
	for op := range m.ops {
		op(factories, consumers)
	}
}

// Start all the consumers from factories
func (m *Manager) Start(fs []Factory) error {
	var err error
	var wg sync.WaitGroup
	wg.Add(1)
	m.ops <- func(factories map[string]Factory, consumers map[string]Consumer) {
		defer wg.Done()
		for _, f := range fs {
			factories[f.Name()] = f
			var cs []Consumer
			cs, err = f.CreateConsumers()
			if err != nil {
				return
			}
			for _, c := range cs {
				consumers[c.Name()] = c
			}
		}
		for _, c := range consumers {
			c.Run()
		}
	}
	wg.Wait()
	return err
}

// Stop all the consumers
func (m *Manager) Stop() error {
	var errors MultiError
	var wg sync.WaitGroup
	wg.Add(1)
	m.ops <- func(factories map[string]Factory, consumers map[string]Consumer) {
		defer wg.Done()
		for name, c := range consumers {
			err := c.Kill()
			if err != nil {
				errors = append(errors, err)
			}
			delete(consumers, name)
		}
		for name := range factories {
			delete(factories, name)
		}
	}
	wg.Wait()
	if len(errors) > 0 {
		return errors
	}
	return nil
}

func (m *Manager) checkConsumers() {
	tick := time.Tick(m.checkAliveness)
	for range tick {
		m.ops <- func(factories map[string]Factory, consumers map[string]Consumer) {
			for name, c := range consumers {
				if !c.Alive() {
					err := c.Kill() //? we really need to kill a consumer already dead?
					if err != nil {
						//TODO: ADD logs
					}
					delete(consumers, name)
					f, ok := factories[c.FactoryName()]
					if !ok {
						//TODO: add log, for some reason the factory didn't exist anymore?
						continue
					}
					c, err := f.CreateConsumer(name)
					if err != nil {
						//TODO: ADD logs
						continue
					}
					consumers[c.Name()] = c
					c.Run()
				}
			}
		}
	}
}
