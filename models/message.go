package models

import "time"

// ChannelType represents supported messaging platforms.
type ChannelType string

const (
	ChannelEmail    ChannelType = "email"
	ChannelWhatsApp ChannelType = "whatsapp"
	ChannelDiscord  ChannelType = "discord"
	ChannelInternal ChannelType = "internal"
)

// MessageDirection indicates if a message was received or sent.
type MessageDirection string

const (
	DirectionInbound  MessageDirection = "inbound"
	DirectionOutbound MessageDirection = "outbound"
)

// Message represents a unified cross-platform message.
type Message struct {
	ID        string            `json:"id"`
	Channel   ChannelType       `json:"channel"`
	Direction MessageDirection  `json:"direction"`
	Sender    string            `json:"sender"`
	Recipient string            `json:"recipient"`
	Subject   string            `json:"subject,omitempty"`
	Body      string            `json:"body"`
	Read      bool              `json:"read"`
	Timestamp string            `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Account represents a messaging provider connection.
type Account struct {
	ID        string            `json:"id"`
	Type      ChannelType       `json:"type"`
	Name      string            `json:"name"`
	Enabled   bool              `json:"enabled"`
	Address   string            `json:"address"` // e.g. email address or bot name
	Settings  map[string]string `json:"settings,omitempty"`
	UpdatedAt string            `json:"updated_at"`
}

// MessageFilter defines search and pagination parameters.
type MessageFilter struct {
	Channel   ChannelType
	Direction MessageDirection
	UnreadOnly bool
	Limit     int
}

// NewMessage initializes a standardized message record.
func NewMessage(channel ChannelType, dir MessageDirection, sender, recipient, subject, body string) Message {
	return Message{
		ID:        "", // Assigned by broker or storage
		Channel:   channel,
		Direction: dir,
		Sender:    sender,
		Recipient: recipient,
		Subject:   subject,
		Body:      body,
		Read:      dir == DirectionOutbound, // Outbound messages are read by default
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Metadata:  make(map[string]string),
	}
}
