package supervisor

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/leandro-lugaresi/hub"
	"github.com/stretchr/testify/assert"
)

func TestManager(t *testing.T) {
	factories := []Factory{
		newStubFactory("RabbitMQ", 4),
		newStubFactory("NATS", 3),
	}
	manager := NewManager(50*time.Millisecond, hub.New())
	t.Run("Should start all consumers from factories", func(t *testing.T) {
		err := manager.Start(factories)
		assert.NoError(t, err, "Start should not return an error")
		var wg sync.WaitGroup
		wg.Add(1)
		manager.ops <- func(factories map[string]Factory, consumers map[string]Consumer) {
			assert.Len(t, factories, 2, "Should be 2 factories inside this manager")
			assert.Len(t, consumers, 7, "should be 7 consumers inside this manager")
			for _, c := range consumers {
				assert.True(t, c.(*stubConsumer).runCalled, "Consumer Run should be called")
			}
			wg.Done()
		}
		wg.Wait()
	})
	t.Run("Should checkConsumers for dead consumers", func(t *testing.T) {
		// Simulate a closed connection
		manager.ops <- func(factories map[string]Factory, _ map[string]Consumer) {
			factories["RabbitMQ"].(*stubFactory).Close()
		}
		time.Sleep(5 * time.Millisecond)
		manager.ops <- func(factories map[string]Factory, consumers map[string]Consumer) {
			for name, c := range consumers {
				if strings.HasPrefix(name, "RabbitMQ-") {
					assert.False(t, c.Alive(), "The %s should be dead", name)
				}
			}
			factories["RabbitMQ"].(*stubFactory).Reconnect()
		}
		time.Sleep(50 * time.Millisecond) //sleep to checkConsumer run
		var wg sync.WaitGroup
		wg.Add(1)
		manager.ops <- func(_ map[string]Factory, consumers map[string]Consumer) {
			for name, c := range consumers {
				if strings.HasPrefix(name, "RabbitMQ-") {
					assert.True(t, c.Alive(), "The %s should be alive now", name)
					assert.True(t, c.(*stubConsumer).runCalled, "Consumer Run should be called")
				}
			}
			wg.Done()
		}
		wg.Wait()
	})
	t.Run("Stop should stop all consumers", func(t *testing.T) {
		manager.Stop()
		var wg sync.WaitGroup
		wg.Add(1)
		manager.ops <- func(factories map[string]Factory, consumers map[string]Consumer) {
			assert.Len(t, factories, 0, "Should be 0 factories inside this manager")
			assert.Len(t, consumers, 0, "should be 0 consumers inside this manager")
			wg.Done()
		}
		wg.Wait()
	})
}
