package collectors

import (
	"context"
	"fmt"
	"sync"

	"github.com/benzjeremy/omnichannel-hub/broker"
	"github.com/benzjeremy/omnichannel-hub/models"
)

// Collector defines standard operations for external communication channels.
type Collector interface {
	ID() string
	Type() models.ChannelType
	Start(ctx context.Context) error
	Stop() error
	Send(msg *models.Message) error
}

// Manager orchestrates lifecycle and dispatching across all active collectors.
type Manager struct {
	mu         sync.RWMutex
	broker     *broker.Broker
	collectors map[models.ChannelType]Collector
}

// NewManager creates an initialized collectors manager.
func NewManager(b *broker.Broker) *Manager {
	return &Manager{
		broker:     b,
		collectors: make(map[models.ChannelType]Collector),
	}
}

// Register registers a new channel collector.
func (m *Manager) Register(c Collector) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.collectors[c.Type()] = c
}

// GetCollector retrieves a collector by channel type.
func (m *Manager) GetCollector(t models.ChannelType) (Collector, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.collectors[t]
	return c, ok
}

// StartAll starts all registered collectors.
func (m *Manager) StartAll(ctx context.Context) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, c := range m.collectors {
		go c.Start(ctx)
	}
}

// StopAll terminates all running collectors.
func (m *Manager) StopAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, c := range m.collectors {
		c.Stop()
	}
}

// Send routes an outgoing message through the appropriate channel collector.
func (m *Manager) Send(msg *models.Message) error {
	m.mu.RLock()
	c, ok := m.collectors[msg.Channel]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("no collector registered for channel type: %s", msg.Channel)
	}

	msg.Direction = models.DirectionOutbound
	msg.Read = true

	// Send through collector
	if err := c.Send(msg); err != nil {
		return fmt.Errorf("failed sending via %s: %w", msg.Channel, err)
	}

	// Publish to broker to record and inform frontend
	return m.broker.Publish(msg)
}

// ListChannels returns all registered channel types.
func (m *Manager) ListChannels() []models.ChannelType {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var types []models.ChannelType
	for t := range m.collectors {
		types = append(types, t)
	}
	return types
}
