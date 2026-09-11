package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/benzjeremy/omnichannel-hub/broker"
	"github.com/benzjeremy/omnichannel-hub/collectors"
	"github.com/benzjeremy/omnichannel-hub/models"
	"github.com/benzjeremy/omnichannel-hub/storage"
)

func setupTestServer(t *testing.T) (*Server, *httptest.Server, func()) {
	tmpDir, _ := os.MkdirTemp("", "api_hub_test_*")
	vault, _ := storage.OpenVault(filepath.Join(tmpDir, "vault.enc"), "APITestSecret123!")
	brk := broker.NewBroker(vault)
	mgr := collectors.NewManager(brk)

	// Register collectors
	mgr.Register(collectors.NewEmailCollector("email-1", "test@benzjeremy.de", brk))
	mgr.Register(collectors.NewDiscordCollector("discord-1", "HubBot", brk))

	cfg := ServerConfig{
		Port:    0,
		Token:   "super-secure-fixed-32-byte-token-2026",
		Version: "v1.0",
	}

	srv, err := NewServer(cfg, vault, brk, mgr)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/messages", srv.handleMessages)
	mux.HandleFunc("/messages/send", srv.handleSend)
	mux.HandleFunc("/messages/read", srv.handleMarkRead)
	mux.HandleFunc("/channels", srv.handleChannels)
	mux.HandleFunc("/stats", srv.handleStats)

	handler := srv.securityMiddleware(mux)
	ts := httptest.NewServer(handler)

	cleanup := func() {
		ts.Close()
		os.RemoveAll(tmpDir)
	}

	return srv, ts, cleanup
}

func TestHealthEndpoint(t *testing.T) {
	_, ts, cleanup := setupTestServer(t)
	defer cleanup()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
	}
}

func TestTokenAndHeadersSecurity(t *testing.T) {
	_, ts, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. Without Token -> 401
	resp, err := http.Get(ts.URL + "/messages")
	if err != nil {
		t.Fatalf("GET /messages failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected 401 without token, got %d", resp.StatusCode)
	}

	// 2. With valid token
	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/messages", nil)
	req.Header.Set("X-Hub-Token", "super-secure-fixed-32-byte-token-2026")

	resp2, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET /messages with token failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK with token, got %d", resp2.StatusCode)
	}
	if resp2.Header.Get("X-Frame-Options") != "DENY" || resp2.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("Missing security headers: %v", resp2.Header)
	}
}

func TestSendAndMarkRead(t *testing.T) {
	_, ts, cleanup := setupTestServer(t)
	defer cleanup()

	client := &http.Client{}

	// Send outbound email
	sendPayload := map[string]interface{}{
		"channel":   "email",
		"recipient": "user@domain.com",
		"subject":   "Projekt Update",
		"body":      "Das System läuft stabil.",
	}
	payloadBytes, _ := json.Marshal(sendPayload)

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/messages/send", bytes.NewReader(payloadBytes))
	req.Header.Set("X-Hub-Token", "super-secure-fixed-32-byte-token-2026")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST /messages/send failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK on send, got %d", resp.StatusCode)
	}

	// Retrieve messages
	reqGet, _ := http.NewRequest(http.MethodGet, ts.URL+"/messages", nil)
	reqGet.Header.Set("X-Hub-Token", "super-secure-fixed-32-byte-token-2026")

	respGet, err := client.Do(reqGet)
	if err != nil {
		t.Fatalf("GET /messages failed: %v", err)
	}
	defer respGet.Body.Close()

	var resData struct {
		Count    int              `json:"count"`
		Messages []models.Message `json:"messages"`
	}
	json.NewDecoder(respGet.Body).Decode(&resData)

	if resData.Count != 1 || resData.Messages[0].Recipient != "user@domain.com" {
		t.Errorf("Unexpected messages list: %+v", resData)
	}
}
