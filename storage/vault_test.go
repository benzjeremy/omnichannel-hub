package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/benzjeremy/omnichannel-hub/models"
)

func TestOmnichannelVault(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "hub_vault_test_*")
	defer os.RemoveAll(tmpDir)

	vaultPath := filepath.Join(tmpDir, "test_hub.enc")
	passphrase := "HyperSecureZeroDummyHubKey2026!"

	v, err := OpenVault(vaultPath, passphrase)
	if err != nil {
		t.Fatalf("OpenVault failed: %v", err)
	}

	// 1. Add Messages
	msg1 := models.NewMessage(models.ChannelEmail, models.DirectionInbound, "alice@example.com", "benzjeremy@pm.me", "Support Anfrage", "Hallo Jeremy, super Projekt!")
	msg2 := models.NewMessage(models.ChannelDiscord, models.DirectionInbound, "BenzoFan#1337", "#general", "", "Wann kommt der nächste Stream?")

	if err := v.SaveMessage(msg1); err != nil {
		t.Fatalf("SaveMessage 1 failed: %v", err)
	}
	if err := v.SaveMessage(msg2); err != nil {
		t.Fatalf("SaveMessage 2 failed: %v", err)
	}

	// 2. Add Account
	acc := models.Account{
		ID:      "acc-email-1",
		Type:    models.ChannelEmail,
		Name:    "ProtonMail Support",
		Enabled: true,
		Address: "benzjeremy@pm.me",
	}
	if err := v.SaveAccount(acc); err != nil {
		t.Fatalf("SaveAccount failed: %v", err)
	}

	// 3. Re-open and verify persistence
	v2, err := OpenVault(vaultPath, passphrase)
	if err != nil {
		t.Fatalf("Re-opening vault failed: %v", err)
	}

	msgs := v2.GetMessages(models.MessageFilter{})
	if len(msgs) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(msgs))
	}

	// Filter unread
	unread := v2.GetMessages(models.MessageFilter{UnreadOnly: true})
	if len(unread) != 2 {
		t.Errorf("Expected 2 unread messages, got %d", len(unread))
	}

	// Mark read
	if err := v2.MarkRead(msgs[0].ID); err != nil {
		t.Fatalf("MarkRead failed: %v", err)
	}

	unreadAfter := v2.GetMessages(models.MessageFilter{UnreadOnly: true})
	if len(unreadAfter) != 1 {
		t.Errorf("Expected 1 unread message after MarkRead, got %d", len(unreadAfter))
	}

	// Check accounts
	accounts := v2.GetAccounts()
	if len(accounts) != 1 || accounts[0].Address != "benzjeremy@pm.me" {
		t.Errorf("Account data mismatch: %v", accounts)
	}
}
