package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
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

// Publish sends the message to the configured Discord Webhook URL.
// If the message exceeds Discord's 2000 character limit, it is split into
// multiple messages at line boundaries (category boundaries).
func (p *WebhookPublisher) Publish(message string) error {
	chunks := splitMessage(message, 2000)
	log.Printf("Sending %d message(s) to Discord Webhook", len(chunks))

	for i, chunk := range chunks {
		if err := p.send(chunk); err != nil {
			return fmt.Errorf("failed to send discord chunk %d/%d: %w", i+1, len(chunks), err)
		}
	}

	log.Println("Discord message(s) sent successfully")
	return nil
}

// send delivers a single chunk to the Discord webhook.
func (p *WebhookPublisher) send(content string) error {
	reqBody := WebhookRequest{
		Content: content,
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

	return nil
}

// splitMessage splits a message into chunks of at most maxLen characters,
// splitting at line boundaries so categories are not cut mid-line.
// If a single line exceeds maxLen, it is split forcibly.
func splitMessage(message string, maxLen int) []string {
	if len(message) <= maxLen {
		return []string{message}
	}

	lines := strings.Split(message, "\n")
	var chunks []string
	var current []string
	currentLen := 0

	for _, line := range lines {
		// If a single line exceeds maxLen, flush the current chunk and split the line.
		if len(line) > maxLen {
			if len(current) > 0 {
				chunks = append(chunks, strings.Join(current, "\n"))
				current = nil
				currentLen = 0
			}
			for len(line) > maxLen {
				chunks = append(chunks, line[:maxLen])
				line = line[maxLen:]
			}
			current = []string{line}
			currentLen = len(line)
			continue
		}

		// +1 for the newline between lines (except the first line in a chunk)
		addLen := len(line)
		if len(current) > 0 {
			addLen++ // for \n
		}

		if currentLen+addLen > maxLen && len(current) > 0 {
			chunks = append(chunks, strings.Join(current, "\n"))
			current = []string{line}
			currentLen = len(line)
		} else {
			current = append(current, line)
			currentLen += addLen
		}
	}

	if len(current) > 0 {
		chunks = append(chunks, strings.Join(current, "\n"))
	}

	return chunks
}
