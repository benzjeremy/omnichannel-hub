package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/benzjeremy/omnichannel-hub/broker"
	"github.com/benzjeremy/omnichannel-hub/collectors"
	"github.com/benzjeremy/omnichannel-hub/models"
	"github.com/benzjeremy/omnichannel-hub/storage"
)

// ServerConfig holds API parameters.
type ServerConfig struct {
	Port    int
	Token   string
	Version string
}

// Server provides the secured unified messaging HTTP interface.
type Server struct {
	config   ServerConfig
	token    string
	vault    *storage.Vault
	broker   *broker.Broker
	manager  *collectors.Manager
	server   *http.Server
	listener net.Listener
}

// NewServer initializes a new secured API server.
func NewServer(cfg ServerConfig, v *storage.Vault, b *broker.Broker, m *collectors.Manager) (*Server, error) {
	tok := cfg.Token
	if tok == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return nil, fmt.Errorf("failed generating CSPRNG token: %w", err)
		}
		tok = hex.EncodeToString(b)
	}

	if cfg.Port <= 0 {
		cfg.Port = 8082
	}
	if cfg.Version == "" {
		cfg.Version = "v1.0"
	}

	return &Server{
		config:  cfg,
		token:   tok,
		vault:   v,
		broker:  b,
		manager: m,
	}, nil
}

// Token returns the active 32-byte security token.
func (s *Server) Token() string {
	return s.token
}

// Start begins serving on 127.0.0.1:port.
func (s *Server) Start() error {
	addr := fmt.Sprintf("127.0.0.1:%d", s.config.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to bind to %s: %w", addr, err)
	}
	s.listener = ln

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/messages", s.handleMessages)
	mux.HandleFunc("/messages/send", s.handleSend)
	mux.HandleFunc("/messages/read", s.handleMarkRead)
	mux.HandleFunc("/channels", s.handleChannels)
	mux.HandleFunc("/stats", s.handleStats)

	handler := s.securityMiddleware(mux)

	s.server = &http.Server{
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	go s.server.Serve(ln)
	return nil
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// Addr returns the listening address.
func (s *Server) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return fmt.Sprintf("127.0.0.1:%d", s.config.Port)
}

func (s *Server) securityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Mandatory Security Headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'")

		// 2. Anti-DNS-Rebinding
		host := r.Host
		if strings.Contains(host, ":") {
			host, _, _ = net.SplitHostPort(host)
		}
		if host != "127.0.0.1" && host != "localhost" && host != "" {
			http.Error(w, `{"error":"forbidden: dns rebinding protection"}`, http.StatusForbidden)
			return
		}

		// 3. Anti-CSRF
		origin := r.Header.Get("Origin")
		if origin != "" && origin != "null" {
			if !strings.HasPrefix(origin, "http://127.0.0.1") && !strings.HasPrefix(origin, "http://localhost") {
				http.Error(w, `{"error":"forbidden: cross-origin request blocked"}`, http.StatusForbidden)
				return
			}
		}

		// 4. Token validation (health exempt)
		if r.URL.Path != "/health" {
			authHeader := r.Header.Get("X-Hub-Token")
			if authHeader == "" {
				authHeader = r.URL.Query().Get("token")
			}
			if authHeader != s.token {
				http.Error(w, `{"error":"unauthorized: invalid or missing security token"}`, http.StatusUnauthorized)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"service":   "omnichannel-hub",
		"version":   s.config.Version,
		"channels":  s.manager.ListChannels(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	filter := models.MessageFilter{
		Channel:    models.ChannelType(q.Get("channel")),
		Direction:  models.MessageDirection(q.Get("direction")),
		UnreadOnly: q.Get("unread") == "true",
	}

	if limitStr := q.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}

	msgs := s.vault.GetMessages(filter)
	if msgs == nil {
		msgs = make([]models.Message, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":    len(msgs),
		"messages": msgs,
	})
}

type SendRequest struct {
	Channel   models.ChannelType `json:"channel"`
	Recipient string             `json:"recipient"`
	Subject   string             `json:"subject,omitempty"`
	Body      string             `json:"body"`
	Sender    string             `json:"sender,omitempty"`
}

func (s *Server) handleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req SendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json body"}`, http.StatusBadRequest)
		return
	}

	if req.Channel == "" || req.Recipient == "" || req.Body == "" {
		http.Error(w, `{"error":"channel, recipient and body are required"}`, http.StatusBadRequest)
		return
	}

	sender := req.Sender
	if sender == "" {
		sender = "Jeremy Benz (Hub)"
	}

	msg := models.NewMessage(req.Channel, models.DirectionOutbound, sender, req.Recipient, req.Subject, req.Body)
	if err := s.manager.Send(&msg); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "dispatched",
		"message": msg,
	})
}

type MarkReadRequest struct {
	ID string `json:"id"`
}

func (s *Server) handleMarkRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req MarkReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json body"}`, http.StatusBadRequest)
		return
	}

	if err := s.vault.MarkRead(req.ID); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Message marked as read",
	})
}

func (s *Server) handleChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"active_channels": s.manager.ListChannels(),
			"accounts":        s.vault.GetAccounts(),
		})
		return
	}

	if r.Method == http.MethodPost {
		var acc models.Account
		if err := json.NewDecoder(r.Body).Decode(&acc); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}
		if acc.ID == "" {
			acc.ID = fmt.Sprintf("acc-%d", time.Now().UnixNano())
		}
		if err := s.vault.SaveAccount(acc); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed saving account: %v"}`, err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "saved",
			"account": acc,
		})
		return
	}

	http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"channel_stats": s.broker.GetStats(),
	})
}
