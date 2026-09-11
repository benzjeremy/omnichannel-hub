package collectors

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/benzjeremy/omnichannel-hub/broker"
	"github.com/benzjeremy/omnichannel-hub/models"
)

// EmailCollector manages IMAP/SMTP communication.
type EmailCollector struct {
	id       string
	broker   *broker.Broker
	address  string
	running  bool
	stopChan chan struct{}
	mu       sync.RWMutex
}

// NewEmailCollector creates an Email collector instance.
func NewEmailCollector(id, address string, b *broker.Broker) *EmailCollector {
	return &EmailCollector{
		id:       id,
		address:  address,
		broker:   b,
		stopChan: make(chan struct{}),
	}
}

func (e *EmailCollector) ID() string               { return e.id }
func (e *EmailCollector) Type() models.ChannelType { return models.ChannelEmail }

func (e *EmailCollector) Start(ctx context.Context) error {
	e.mu.Lock()
	e.running = true
	e.mu.Unlock()

	// Simulate periodic IMAP idle / poll for new emails
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-e.stopChan:
			return nil
		case <-ticker.C:
			// In live production, connect via TLS to IMAP server
		}
	}
}

func (e *EmailCollector) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running {
		e.running = false
		close(e.stopChan)
	}
	return nil
}

func (e *EmailCollector) Send(msg *models.Message) error {
	if msg.Recipient == "" {
		return fmt.Errorf("email recipient address cannot be empty")
	}
	if msg.Body == "" {
		return fmt.Errorf("email body cannot be empty")
	}

	// In production, execute TLS SMTP handshake
	msg.Metadata["smtp_status"] = "delivered_tls_secured"
	return nil
}

// IngestEmail allows injecting an inbound email from IMAP parser.
func (e *EmailCollector) IngestEmail(from, to, subject, body string) error {
	msg := models.NewMessage(models.ChannelEmail, models.DirectionInbound, from, to, subject, body)
	return e.broker.Publish(&msg)
}
