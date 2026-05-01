package discord

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDiscordPublish(t *testing.T) {
	testMessage := "Hello from YNAB Weekly Wrap!"

	// Create a mock server to receive the webhook request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Verify content type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected application/json content type, got %s", r.Header.Get("Content-Type"))
		}

		// Read and verify body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read request body: %v", err)
		}

		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("Failed to unmarshal request body: %v", err)
		}

		if payload["content"] != testMessage {
			t.Errorf("Expected content %q, got %q", testMessage, payload["content"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Initialize publisher with mock server URL
	p := &WebhookPublisher{
		WebhookURL: server.URL,
	}

	// Test Publish
	err := p.Publish(testMessage)
	if err != nil {
		t.Errorf("Publish failed: %v", err)
	}
}

func TestDiscordPublish_Split(t *testing.T) {
	// Build a long message with newlines (realistic category lines) that exceeds 2000 chars.
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, fmt.Sprintf("Line %d: abcdefghijklmnopqrstuvwxyz1234567890", i))
	}
	longMessage := strings.Join(lines, "\n")

	if len(longMessage) <= 2000 {
		t.Fatalf("Test message should exceed 2000 chars, got %d", len(longMessage))
	}

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]string
		_ = json.Unmarshal(body, &payload)

		if len(payload["content"]) > 2000 {
			t.Errorf("Expected chunk length <= 2000, got %d", len(payload["content"]))
		}

		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := &WebhookPublisher{
		WebhookURL: server.URL,
	}

	err := p.Publish(longMessage)
	if err != nil {
		t.Errorf("Publish failed: %v", err)
	}

	if requestCount < 2 {
		t.Errorf("Expected multiple requests for split message, got %d", requestCount)
	}
}

func TestDiscordPublish_SplitAtLineBoundary(t *testing.T) {
	// Build a message where categories are lines; the split should occur at line boundaries
	var parts []string
	for i := 0; i < 50; i++ {
		parts = append(parts, fmt.Sprintf("Category %d: this is a spending line with some extra text", i))
	}
	message := strings.Join(parts, "\n")

	if len(message) <= 2000 {
		t.Fatalf("Test message should exceed 2000 chars, got %d", len(message))
	}

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]string
		_ = json.Unmarshal(body, &payload)

		if len(payload["content"]) > 2000 {
			t.Errorf("Expected chunk length <= 2000, got %d", len(payload["content"]))
		}

		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := &WebhookPublisher{
		WebhookURL: server.URL,
	}

	err := p.Publish(message)
	if err != nil {
		t.Errorf("Publish failed: %v", err)
	}

	if requestCount < 2 {
		t.Errorf("Expected multiple requests for split message, got %d", requestCount)
	}
}
