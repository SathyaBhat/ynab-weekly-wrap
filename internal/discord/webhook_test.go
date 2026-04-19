package discord

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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

func TestDiscordPublish_Truncate(t *testing.T) {
	// 2005 characters (exceeds 2000)
	longMessage := ""
	for i := 0; i < 201; i++ {
		longMessage += "1234567890"
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]string
		_ = json.Unmarshal(body, &payload)

		if len(payload["content"]) > 2000 {
			t.Errorf("Expected truncated content (max 2000), got length %d", len(payload["content"]))
		}

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
}
