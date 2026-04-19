package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// WebhookPublisher implements the publisher.Publisher interface for Discord Webhooks
type WebhookPublisher struct {
	WebhookURL string
}

// WebhookRequest represents the payload for Discord Webhook
type WebhookRequest struct {
	Content string `json:"content"`
}

// NewWebhookPublisher creates a new Discord Webhook publisher
func NewWebhookPublisher(webhookURL string) *WebhookPublisher {
	return &WebhookPublisher{
		WebhookURL: webhookURL,
	}
}

// Publish sends the message to the configured Discord Webhook URL
func (p *WebhookPublisher) Publish(message string) error {
	log.Printf("Sending message to Discord Webhook")

	// Discord has a 2000 character limit for content
	if len(message) > 2000 {
		log.Printf("Message too long for Discord (%d characters), truncating to 2000", len(message))
		message = message[:1997] + "..."
	}

	reqBody := WebhookRequest{
		Content: message,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal discord request: %w", err)
	}

	resp, err := http.Post(p.WebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send discord request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord API error (status %d): %s", resp.StatusCode, string(body))
	}

	log.Println("Discord message sent successfully")
	return nil
}
