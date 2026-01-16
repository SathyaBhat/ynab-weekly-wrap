package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	YNAB           YNABConfig           `yaml:"ynab"`
	Telegram       TelegramConfig       `yaml:"telegram"`
	Schedule       ScheduleConfig       `yaml:"schedule"`
	Logging        LoggingConfig        `yaml:"logging"`
	Thresholds     ThresholdConfig      `yaml:"thresholds"`
	WeeklyAnalysis WeeklyAnalysisConfig `yaml:"weekly_analysis"`
}

type YNABConfig struct {
	APIToken string `yaml:"api_token"`
	BudgetID string `yaml:"budget_id"`
	BaseURL  string `yaml:"base_url"`
}

type TelegramConfig struct {
	BotToken string `yaml:"bot_token"`
	ChatID   int64  `yaml:"chat_id"`
	TopicID  int    `yaml:"topic_id"` // Optional: Topic ID for topics in supergroups
}

type ScheduleConfig struct {
	Cron     string `yaml:"cron"`
	Timezone string `yaml:"timezone"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type ThresholdConfig struct {
	AtRiskPercent      int `yaml:"at_risk_percent"`
	OverBudgetPercent  int `yaml:"over_budget_percent"`
	TopCategoriesCount int `yaml:"top_categories_count"`
}

type WeeklyAnalysisConfig struct {
	IncludeOffBudget  bool     `yaml:"include_off_budget"`
	IncludeTransfers  bool     `yaml:"include_transfers"`
	FocusCategories   []string `yaml:"focus_categories"`
	ExcludeCategories []string `yaml:"exclude_categories"`
}

// loadEnvFile loads environment variables from a .env file
func loadEnvFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		// File doesn't exist, that's okay
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		// Parse KEY=VALUE format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) >= 2 && (value[0] == '"' && value[len(value)-1] == '"') {
			value = value[1 : len(value)-1]
		}

		// Set environment variable
		os.Setenv(key, value)
	}

	return scanner.Err()
}

func LoadConfig() (*Config, error) {
	// Load .env file if it exists (optional)
	_ = loadEnvFile(".env")
	_ = loadEnvFile("/app/.env")
	config := &Config{}

	// Load from environment variables
	config.YNAB.APIToken = os.Getenv("YNAB_API_TOKEN")
	config.YNAB.BudgetID = os.Getenv("YNAB_BUDGET_ID")

	config.Telegram.BotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	if chatIDStr := os.Getenv("TELEGRAM_CHAT_ID"); chatIDStr != "" {
		if chatID, err := strconv.ParseInt(chatIDStr, 10, 64); err == nil {
			config.Telegram.ChatID = chatID
		}
	}
	if topicIDStr := os.Getenv("TELEGRAM_TOPIC_ID"); topicIDStr != "" {
		if topicID, err := strconv.Atoi(topicIDStr); err == nil {
			config.Telegram.TopicID = topicID
		}
	}

	config.Schedule.Cron = os.Getenv("SCHEDULE_CRON")
	config.Logging.Level = os.Getenv("LOG_LEVEL")
	if topCategoriesStr := os.Getenv("TOP_CATEGORIES_COUNT"); topCategoriesStr != "" {
		if count, err := strconv.Atoi(topCategoriesStr); err == nil {
			config.Thresholds.TopCategoriesCount = count
		}
	}

	// Set defaults
	if config.Schedule.Cron == "" {
		config.Schedule.Cron = "0 9 * * 1"
	}
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	if config.Logging.Format == "" {
		config.Logging.Format = "json"
	}
	if config.Thresholds.AtRiskPercent == 0 {
		config.Thresholds.AtRiskPercent = 75
	}
	if config.Thresholds.OverBudgetPercent == 0 {
		config.Thresholds.OverBudgetPercent = 100
	}

	return config, nil
}

// ValidateConfig validates required configuration fields
// testMode: if true, skip Telegram validation (useful for dry-run testing)
func ValidateConfig(config *Config, testMode bool) error {
	// Always require YNAB credentials
	if config.YNAB.APIToken == "" {
		return fmt.Errorf("YNAB API token is required (set YNAB_API_TOKEN)")
	}
	if config.YNAB.BudgetID == "" {
		return fmt.Errorf("YNAB budget ID is required (set YNAB_BUDGET_ID)")
	}

	// In test mode (dry-run), skip Telegram validation
	if testMode {
		return nil
	}

	// For production, require Telegram credentials
	if config.Telegram.BotToken == "" {
		return fmt.Errorf("Telegram bot token is required (set TELEGRAM_BOT_TOKEN)")
	}
	if config.Telegram.ChatID == 0 {
		return fmt.Errorf("Telegram chat ID is required (set TELEGRAM_CHAT_ID)")
	}
	return nil
}
