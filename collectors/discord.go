package collectors

import (
	"context"
	"fmt"
	"sync"

	"github.com/benzjeremy/omnichannel-hub/broker"
	"github.com/benzjeremy/omnichannel-hub/models"
)

// DiscordCollector interfaces with Discord Bot API / Webhooks.
type DiscordCollector struct {
	id       string
	broker   *broker.Broker
	botName  string
	running  bool
	stopChan chan struct{}
	mu       sync.RWMutex
}

// NewDiscordCollector initializes a Discord collector.
func NewDiscordCollector(id, botName string, b *broker.Broker) *DiscordCollector {
	return &DiscordCollector{
		id:       id,
		botName:  botName,
		broker:   b,
		stopChan: make(chan struct{}),
	}
}

func (d *DiscordCollector) ID() string               { return d.id }
func (d *DiscordCollector) Type() models.ChannelType { return models.ChannelDiscord }

func (d *DiscordCollector) Start(ctx context.Context) error {
	d.mu.Lock()
	d.running = true
	d.mu.Unlock()

	select {
	case <-ctx.Done():
		return nil
	case <-d.stopChan:
		return nil
	}
}

func (d *DiscordCollector) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.running {
		d.running = false
		close(d.stopChan)
	}
	return nil
}

func (d *DiscordCollector) Send(msg *models.Message) error {
	if msg.Recipient == "" {
		return fmt.Errorf("discord channel or user ID cannot be empty")
	}
	if msg.Body == "" {
		return fmt.Errorf("message content cannot be empty")
	}

	msg.Metadata["discord_status"] = "dispatched_to_gateway"
	return nil
}

// IngestDiscordMessage injects a received Discord gateway event.
func (d *DiscordCollector) IngestDiscordMessage(author, channelID, content string) error {
	msg := models.NewMessage(models.ChannelDiscord, models.DirectionInbound, author, channelID, "", content)
	return d.broker.Publish(&msg)
}
