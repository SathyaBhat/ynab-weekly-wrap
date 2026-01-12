package telegram

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sathyabhat/ynab-weekly-wrap/internal/config"
)

type Bot struct {
	bot    *tgbotapi.BotAPI
	config config.TelegramConfig
}

func NewBot(telegramConfig config.TelegramConfig) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(telegramConfig.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Telegram bot: %w", err)
	}

	return &Bot{
		bot:    bot,
		config: telegramConfig,
	}, nil
}

func (b *Bot) SendWeeklyWrap(message string) error {
	log.Printf("Sending weekly wrap to chat ID: %d", b.config.ChatID)

	msg := tgbotapi.NewMessage(b.config.ChatID, message)
	msg.ParseMode = "Markdown"

	_, err := b.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send Telegram message: %w", err)
	}

	log.Println("Weekly wrap sent successfully")
	return nil
}

func (b *Bot) TestConnection() error {
	_, err := b.bot.GetMe()
	if err != nil {
		return fmt.Errorf("failed to test Telegram bot connection: %w", err)
	}

	log.Println("Telegram bot connection test successful")
	return nil
}
