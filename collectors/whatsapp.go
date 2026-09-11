package collectors

import (
	"context"
	"fmt"
	"sync"

	"github.com/benzjeremy/omnichannel-hub/broker"
	"github.com/benzjeremy/omnichannel-hub/models"
)

// WhatsAppCollector manages WhatsApp messaging via local Baileys bridge.
type WhatsAppCollector struct {
	id       string
	broker   *broker.Broker
	phone    string
	running  bool
	stopChan chan struct{}
	mu       sync.RWMutex
}

// NewWhatsAppCollector initializes a WhatsApp collector.
func NewWhatsAppCollector(id, phone string, b *broker.Broker) *WhatsAppCollector {
	return &WhatsAppCollector{
		id:       id,
		phone:    phone,
		broker:   b,
		stopChan: make(chan struct{}),
	}
}

func (w *WhatsAppCollector) ID() string               { return w.id }
func (w *WhatsAppCollector) Type() models.ChannelType { return models.ChannelWhatsApp }

func (w *WhatsAppCollector) Start(ctx context.Context) error {
	w.mu.Lock()
	w.running = true
	w.mu.Unlock()

	select {
	case <-ctx.Done():
		return nil
	case <-w.stopChan:
		return nil
	}
}

func (w *WhatsAppCollector) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.running {
		w.running = false
		close(w.stopChan)
	}
	return nil
}

func (w *WhatsAppCollector) Send(msg *models.Message) error {
	if msg.Recipient == "" {
		return fmt.Errorf("whatsapp phone number cannot be empty")
	}
	if msg.Body == "" {
		return fmt.Errorf("whatsapp message cannot be empty")
	}

	msg.Metadata["whatsapp_status"] = "queued_baileys_bridge"
	return nil
}

// IngestWhatsAppMessage injects a received WhatsApp message from bridge.
func (w *WhatsAppCollector) IngestWhatsAppMessage(sender, chatID, content string) error {
	msg := models.NewMessage(models.ChannelWhatsApp, models.DirectionInbound, sender, chatID, "", content)
	return w.broker.Publish(&msg)
}
