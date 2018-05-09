package supervisor

import (
	"sync"
	"time"

	"github.com/leandro-lugaresi/hub"
)

// Manager is the block responsible for creating all the consumers.
// Keeping track of the current state of consumers and stop/restart consumers when needed.
type Manager struct {
	hub            *hub.Hub
	checkAliveness time.Duration
	ops            chan func(map[string]Factory, map[string]Consumer)
}

// NewManager init a new manager and wait for operations.
func NewManager(intervalChecks time.Duration, hub *hub.Hub) *Manager {
	m := &Manager{
		hub:            hub,
		checkAliveness: intervalChecks,
		ops:            make(chan func(map[string]Factory, map[string]Consumer)),
	}
	go m.work()
	return m
}

// work will execute all te operations received from the internal operation channel
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

// CheckConsumers will tick and send operations to do some checks
func (m *Manager) CheckConsumers(cancel <-chan struct{}) {
	ticker := time.NewTicker(m.checkAliveness)
	for {
		select {
		case <-ticker.C:
			m.ops <- m.restartDeadConsumers
		case <-cancel:
			return
		}
	}
}

func (m *Manager) restartDeadConsumers(factories map[string]Factory, consumers map[string]Consumer) {
	for name, c := range consumers {
		if !c.Alive() {
			m.hub.Publish(hub.Message{
				Name: "supervisor.recreating_consumer.info",
				Body: []byte("Recreating one consumer"),
				Fields: hub.Fields{
					"factory-name":  c.FactoryName(),
					"consumer-name": name,
				},
			})
			delete(consumers, name)
			f, ok := factories[c.FactoryName()]
			if !ok {
				m.hub.Publish(hub.Message{
					Name: "supervisor.recreating_consumer.warning",
					Body: []byte("Factory did not exist anymore"),
					Fields: hub.Fields{
						"factory-name":  c.FactoryName(),
						"consumer-name": name,
					},
				})
				continue
			}
			nc, err := f.CreateConsumer(name)
			if err != nil {
				m.hub.Publish(hub.Message{
					Name: "supervisor.recreating_consumer.error",
					Body: []byte("Error recreating one consumer"),
					Fields: hub.Fields{
						"factory-name":  c.FactoryName(),
						"consumer-name": name,
						"error":         err,
					},
				})
				continue
			}
			consumers[name] = nc
			nc.Run()
		}
	}
}
