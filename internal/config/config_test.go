package config

import (
	"os"
	"testing"
)

// clearEnv unsets all environment variables used by LoadConfig.
func clearEnv(t *testing.T) {
	t.Helper()
	vars := []string{
		"YNAB_API_TOKEN", "YNAB_BUDGET_ID",
		"TELEGRAM_BOT_TOKEN", "TELEGRAM_CHAT_ID", "TELEGRAM_TOPIC_ID",
		"SCHEDULE_CRON", "MONTHLY_SCHEDULE_CRON",
		"LOG_LEVEL", "TOP_CATEGORIES_COUNT",
	}
	for _, v := range vars {
		os.Unsetenv(v)
	}
}

// ── Defaults ─────────────────────────────────────────────────────────────────

func TestLoadConfig_DefaultWeeklyCron(t *testing.T) {
	clearEnv(t)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Schedule.Cron != "0 9 * * 1" {
		t.Errorf("default weekly cron: got %q, want %q", cfg.Schedule.Cron, "0 9 * * 1")
	}
}

func TestLoadConfig_DefaultMonthlyCron(t *testing.T) {
	clearEnv(t)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Schedule.MonthlyCron != "0 9 1 * *" {
		t.Errorf("default monthly cron: got %q, want %q", cfg.Schedule.MonthlyCron, "0 9 1 * *")
	}
}

func TestLoadConfig_DefaultLogLevel(t *testing.T) {
	clearEnv(t)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("default log level: got %q, want %q", cfg.Logging.Level, "info")
	}
}

func TestLoadConfig_DefaultAtRiskPercent(t *testing.T) {
	clearEnv(t)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Thresholds.AtRiskPercent != 75 {
		t.Errorf("default AtRiskPercent: got %d, want 75", cfg.Thresholds.AtRiskPercent)
	}
}

func TestLoadConfig_DefaultOverBudgetPercent(t *testing.T) {
	clearEnv(t)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Thresholds.OverBudgetPercent != 100 {
		t.Errorf("default OverBudgetPercent: got %d, want 100", cfg.Thresholds.OverBudgetPercent)
	}
}

// ── Env var overrides ─────────────────────────────────────────────────────────

func TestLoadConfig_WeeklyCronOverride(t *testing.T) {
	clearEnv(t)
	os.Setenv("SCHEDULE_CRON", "0 8 * * 5")
	defer os.Unsetenv("SCHEDULE_CRON")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Schedule.Cron != "0 8 * * 5" {
		t.Errorf("weekly cron override: got %q, want %q", cfg.Schedule.Cron, "0 8 * * 5")
	}
}

func TestLoadConfig_MonthlyCronOverride(t *testing.T) {
	clearEnv(t)
	os.Setenv("MONTHLY_SCHEDULE_CRON", "0 7 1 * *")
	defer os.Unsetenv("MONTHLY_SCHEDULE_CRON")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Schedule.MonthlyCron != "0 7 1 * *" {
		t.Errorf("monthly cron override: got %q, want %q", cfg.Schedule.MonthlyCron, "0 7 1 * *")
	}
}

func TestLoadConfig_YNABCredentials(t *testing.T) {
	clearEnv(t)
	os.Setenv("YNAB_API_TOKEN", "tok123")
	os.Setenv("YNAB_BUDGET_ID", "bud456")
	defer func() {
		os.Unsetenv("YNAB_API_TOKEN")
		os.Unsetenv("YNAB_BUDGET_ID")
	}()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.YNAB.APIToken != "tok123" {
		t.Errorf("APIToken: got %q, want %q", cfg.YNAB.APIToken, "tok123")
	}
	if cfg.YNAB.BudgetID != "bud456" {
		t.Errorf("BudgetID: got %q, want %q", cfg.YNAB.BudgetID, "bud456")
	}
}

func TestLoadConfig_TelegramChatID(t *testing.T) {
	clearEnv(t)
	os.Setenv("TELEGRAM_CHAT_ID", "-1001234567890")
	defer os.Unsetenv("TELEGRAM_CHAT_ID")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Telegram.ChatID != -1001234567890 {
		t.Errorf("ChatID: got %d, want -1001234567890", cfg.Telegram.ChatID)
	}
}

func TestLoadConfig_TopCategoriesCount(t *testing.T) {
	clearEnv(t)
	os.Setenv("TOP_CATEGORIES_COUNT", "10")
	defer os.Unsetenv("TOP_CATEGORIES_COUNT")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Thresholds.TopCategoriesCount != 10 {
		t.Errorf("TopCategoriesCount: got %d, want 10", cfg.Thresholds.TopCategoriesCount)
	}
}

// ── ValidateConfig ────────────────────────────────────────────────────────────

func TestValidateConfig_TestMode_MissingYNABToken(t *testing.T) {
	cfg := &Config{}
	err := ValidateConfig(cfg, true)
	if err == nil {
		t.Fatal("expected error for missing YNAB token, got nil")
	}
}

func TestValidateConfig_TestMode_MissingBudgetID(t *testing.T) {
	cfg := &Config{}
	cfg.YNAB.APIToken = "tok"
	err := ValidateConfig(cfg, true)
	if err == nil {
		t.Fatal("expected error for missing BudgetID, got nil")
	}
}

func TestValidateConfig_TestMode_NoTelegramRequired(t *testing.T) {
	cfg := &Config{}
	cfg.YNAB.APIToken = "tok"
	cfg.YNAB.BudgetID = "bud"
	// Telegram fields empty — should be OK in test mode
	err := ValidateConfig(cfg, true)
	if err != nil {
		t.Errorf("unexpected error in test mode: %v", err)
	}
}

func TestValidateConfig_ProductionMode_RequiresTelegram(t *testing.T) {
	cfg := &Config{}
	cfg.YNAB.APIToken = "tok"
	cfg.YNAB.BudgetID = "bud"
	// No Telegram credentials
	err := ValidateConfig(cfg, false)
	if err == nil {
		t.Fatal("expected error for missing Telegram credentials in production mode, got nil")
	}
}

func TestValidateConfig_ProductionMode_Valid(t *testing.T) {
	cfg := &Config{}
	cfg.YNAB.APIToken = "tok"
	cfg.YNAB.BudgetID = "bud"
	cfg.Telegram.BotToken = "bot"
	cfg.Telegram.ChatID = -123
	err := ValidateConfig(cfg, false)
	if err != nil {
		t.Errorf("unexpected error for valid config: %v", err)
	}
}
