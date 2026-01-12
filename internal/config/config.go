package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
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
// This is a simple implementation that parses KEY=VALUE lines
func loadEnvFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		// File doesn't exist, that's okay
		return nil
	}
	defer file.Close()

	count := 0
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
		count++
	}
	

	
	return scanner.Err()
}

func LoadConfig() (*Config, error) {
	// Load .env file if it exists (optional)
	// Check in current directory and common locations
	_ = loadEnvFile(".env")
	_ = loadEnvFile("/app/.env")

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("/app/configs")

	// Enable reading from environment variables
	viper.AutomaticEnv()
	
	// Try to read YAML config file (optional)
	_ = viper.ReadInConfig()

	// Load environment variables with expected names (no prefix)
	// This allows both .env and direct environment variable usage
	viper.BindEnv("ynab.api_token", "YNAB_API_TOKEN")
	viper.BindEnv("ynab.budget_id", "YNAB_BUDGET_ID")
	viper.BindEnv("telegram.bot_token", "TELEGRAM_BOT_TOKEN")
	viper.BindEnv("telegram.chat_id", "TELEGRAM_CHAT_ID")
	viper.BindEnv("schedule.cron", "SCHEDULE_CRON")
	viper.BindEnv("schedule.timezone", "TZ")
	viper.BindEnv("logging.level", "LOG_LEVEL")

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// If environment variables were set and config is still empty, copy from env
	if config.YNAB.APIToken == "" {
		config.YNAB.APIToken = os.Getenv("YNAB_API_TOKEN")
	}
	if config.YNAB.BudgetID == "" {
		config.YNAB.BudgetID = os.Getenv("YNAB_BUDGET_ID")
	}
	if config.Telegram.BotToken == "" {
		config.Telegram.BotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	}
	if config.Telegram.ChatID == 0 {
		chatIDStr := os.Getenv("TELEGRAM_CHAT_ID")
		if chatIDStr != "" {
			var chatID int64
			if _, err := fmt.Sscanf(chatIDStr, "%d", &chatID); err == nil {
				config.Telegram.ChatID = chatID
			}
		}
	}
	if config.Schedule.Cron == "" {
		config.Schedule.Cron = os.Getenv("SCHEDULE_CRON")
	}
	if config.Schedule.Timezone == "" {
		config.Schedule.Timezone = os.Getenv("TZ")
	}
	if config.Logging.Level == "" {
		config.Logging.Level = os.Getenv("LOG_LEVEL")
	}

	// Set defaults
	if config.Schedule.Cron == "" {
		config.Schedule.Cron = "0 9 * * 1" // Monday 9 AM
	}
	if config.Schedule.Timezone == "" {
		config.Schedule.Timezone = "Local"
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
	if config.Thresholds.TopCategoriesCount == 0 {
		config.Thresholds.TopCategoriesCount = 3
	}

	return &config, nil
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
