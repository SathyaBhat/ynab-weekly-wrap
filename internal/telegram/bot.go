package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/sathyabhat/ynab-weekly-wrap/internal/config"
)

type Bot struct {
	config config.TelegramConfig
}

// SendMessageRequest represents the request to send a message via Telegram API
type SendMessageRequest struct {
	ChatID                int64  `json:"chat_id"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode"`
	MessageThreadID       int    `json:"message_thread_id,omitempty"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview"`
}

// APIResponse represents a generic Telegram API response
type APIResponse struct {
	OK    bool        `json:"ok"`
	Error string      `json:"error"`
	Result json.RawMessage `json:"result"`
}

const telegramAPIURL = "https://api.telegram.org"

func NewBot(telegramConfig config.TelegramConfig) (*Bot, error) {
	return &Bot{
		config: telegramConfig,
	}, nil
}

func (b *Bot) SendWeeklyWrap(message string) error {
	log.Printf("Sending weekly wrap to chat ID: %d", b.config.ChatID)

	return b.sendMessage(message)
}

func (b *Bot) sendMessage(message string) error {
	req := SendMessageRequest{
		ChatID:                b.config.ChatID,
		Text:                  message,
		ParseMode:             "Markdown",
		DisableWebPagePreview: true,
	}

	// If topic ID is configured, add it to the request
	if b.config.TopicID > 0 {
		req.MessageThreadID = b.config.TopicID
		log.Printf("Sending message to topic ID: %d", b.config.TopicID)
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", telegramAPIURL, b.config.BotToken)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !apiResp.OK {
		return fmt.Errorf("telegram API error: %s", apiResp.Error)
	}

	log.Println("Weekly wrap sent successfully")
	return nil
}

func (b *Bot) TestConnection() error {
	log.Println("Testing Telegram bot connection...")

	url := fmt.Sprintf("%s/bot%s/getMe", telegramAPIURL, b.config.BotToken)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to test connection: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !apiResp.OK {
		return fmt.Errorf("telegram API error: %s", apiResp.Error)
	}

	log.Println("Telegram bot connection test successful")
	return nil
}
