package broker

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/benzjeremy/omnichannel-hub/models"
	"github.com/benzjeremy/omnichannel-hub/storage"
)

func TestBrokerPubSub(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "broker_test_*")
	defer os.RemoveAll(tmpDir)

	v, _ := storage.OpenVault(filepath.Join(tmpDir, "vault.enc"), "BrokerSecretKey123!")
	brk := NewBroker(v)

	ch, unsub := brk.Subscribe()
	defer unsub()

	msg := models.NewMessage(models.ChannelWhatsApp, models.DirectionInbound, "+491701234567", "Hub", "", "Hallo von WhatsApp!")
	if err := brk.Publish(&msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	select {
	case received := <-ch:
		if received.Body != "Hallo von WhatsApp!" {
			t.Errorf("Unexpected message body: %s", received.Body)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Timeout waiting for message on subscription channel")
	}

	stats := brk.GetStats()
	if stats[models.ChannelWhatsApp].TotalInbound != 1 {
		t.Errorf("Expected 1 inbound whatsapp message in stats, got %d", stats[models.ChannelWhatsApp].TotalInbound)
	}
}
