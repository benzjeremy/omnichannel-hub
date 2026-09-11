package collectors

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/benzjeremy/omnichannel-hub/broker"
	"github.com/benzjeremy/omnichannel-hub/models"
	"github.com/benzjeremy/omnichannel-hub/storage"
)

func TestCollectorsAndManager(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "collectors_test_*")
	defer os.RemoveAll(tmpDir)

	v, _ := storage.OpenVault(filepath.Join(tmpDir, "vault.enc"), "CollectorPass123!")
	brk := broker.NewBroker(v)
	mgr := NewManager(brk)

	// Register Email, Discord and WhatsApp collectors
	emailCol := NewEmailCollector("email-1", "test@benzjeremy.de", brk)
	discordCol := NewDiscordCollector("discord-1", "HubBot#0001", brk)
	waCol := NewWhatsAppCollector("wa-1", "+491701234567", brk)

	mgr.Register(emailCol)
	mgr.Register(discordCol)
	mgr.Register(waCol)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mgr.StartAll(ctx)

	// Check channels list
	channels := mgr.ListChannels()
	if len(channels) != 3 {
		t.Fatalf("Expected 3 channels, got %d", len(channels))
	}

	// Subscribe to broker
	subCh, unsub := brk.Subscribe()
	defer unsub()

	// 1. Inbound test: Ingest Email
	if err := emailCol.IngestEmail("chef@firma.de", "test@benzjeremy.de", "Meeting", "Können wir reden?"); err != nil {
		t.Fatalf("IngestEmail failed: %v", err)
	}

	select {
	case m := <-subCh:
		if m.Channel != models.ChannelEmail || m.Subject != "Meeting" {
			t.Errorf("Unexpected message received: %+v", m)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Timeout waiting for email ingest")
	}

	// 2. Outbound test: Send Discord message
	outMsg := models.NewMessage(models.ChannelDiscord, models.DirectionOutbound, "Me", "channel-42", "", "Hallo Discord!")
	if err := mgr.Send(&outMsg); err != nil {
		t.Fatalf("mgr.Send failed: %v", err)
	}

	select {
	case m := <-subCh:
		if m.Channel != models.ChannelDiscord || m.Body != "Hallo Discord!" {
			t.Errorf("Unexpected outbound message in broker: %+v", m)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Timeout waiting for outbound discord message")
	}

	mgr.StopAll()
}
