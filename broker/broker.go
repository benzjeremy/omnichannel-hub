package broker

import (
	"sync"

	"github.com/benzjeremy/omnichannel-hub/models"
	"github.com/benzjeremy/omnichannel-hub/storage"
)

// Broker handles message routing, fan-out to subscribers, and persistence.
type Broker struct {
	mu          sync.RWMutex
	vault       *storage.Vault
	subscribers map[chan *models.Message]struct{}
	stats       map[models.ChannelType]ChannelStats
}

// ChannelStats tracks metrics per communication channel.
type ChannelStats struct {
	TotalInbound  int `json:"total_inbound"`
	TotalOutbound int `json:"total_outbound"`
}

// NewBroker creates a central message broker.
func NewBroker(v *storage.Vault) *Broker {
	return &Broker{
		vault:       v,
		subscribers: make(map[chan *models.Message]struct{}),
		stats:       make(map[models.ChannelType]ChannelStats),
	}
}

// Publish stores the message safely and broadcasts it to all active listeners.
func (b *Broker) Publish(msg *models.Message) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Persist to encrypted vault
	if err := b.vault.SaveMessage(*msg); err != nil {
		return err
	}

	// Update stats
	s := b.stats[msg.Channel]
	if msg.Direction == models.DirectionInbound {
		s.TotalInbound++
	} else {
		s.TotalOutbound++
	}
	b.stats[msg.Channel] = s

	// Broadcast non-blocking to all subscribers
	for ch := range b.subscribers {
		select {
		case ch <- msg:
		default:
			// Slow consumer drop to avoid stalling broker
		}
	}

	return nil
}

// Subscribe returns a channel receiving all published messages and an unsubscribe callback.
func (b *Broker) Subscribe() (chan *models.Message, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan *models.Message, 100)
	b.subscribers[ch] = struct{}{}

	unsub := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		delete(b.subscribers, ch)
		close(ch)
	}

	return ch, unsub
}

// GetStats returns current message counters.
func (b *Broker) GetStats() map[models.ChannelType]ChannelStats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	copyStats := make(map[models.ChannelType]ChannelStats, len(b.stats))
	for k, v := range b.stats {
		copyStats[k] = v
	}
	return copyStats
}
