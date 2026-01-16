package scheduler

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sathyabhat/ynab-weekly-wrap/internal/config"
	"github.com/sathyabhat/ynab-weekly-wrap/internal/processor"
	"github.com/sathyabhat/ynab-weekly-wrap/internal/telegram"
	"github.com/sathyabhat/ynab-weekly-wrap/internal/ynab"
)

type Scheduler struct {
	cron         *cron.Cron
	config       *config.Config
	ynabClient   *ynab.Client
	telegramBot  *telegram.Bot
	analyzer     *processor.Analyzer
	dryRun       bool
	skipTelegram bool
}

// SchedulerOption is a functional option for configuring Scheduler
type SchedulerOption func(*Scheduler)

// WithDryRun enables dry-run mode (prints to stdout instead of sending to Telegram)
func WithDryRun(dryRun bool) SchedulerOption {
	return func(s *Scheduler) {
		s.dryRun = dryRun
	}
}

// WithSkipTelegram disables Telegram bot creation (for testing without credentials)
func WithSkipTelegram(skip bool) SchedulerOption {
	return func(s *Scheduler) {
		s.skipTelegram = skip
	}
}

func NewScheduler(cfg *config.Config, opts ...SchedulerOption) *Scheduler {
	cronScheduler := cron.New()

	sched := &Scheduler{
		cron:         cronScheduler,
		config:       cfg,
		ynabClient:   ynab.NewClient(cfg.YNAB),
		analyzer:     processor.NewAnalyzer(),
		dryRun:       false,
		skipTelegram: false,
	}

	// Apply options (which may set dryRun or skipTelegram)
	for _, opt := range opts {
		opt(sched)
	}

	// Only create Telegram bot if not skipping it
	if !sched.skipTelegram && !sched.dryRun {
		telegramBot, err := telegram.NewBot(cfg.Telegram)
		if err != nil {
			log.Fatalf("Failed to create Telegram bot: %v", err)
		}
		sched.telegramBot = telegramBot
	}

	return sched
}

func (s *Scheduler) Start() error {
	log.Printf("Starting scheduler with cron expression: %s", s.config.Schedule.Cron)

	// Add weekly wrap job
	_, err := s.cron.AddFunc(s.config.Schedule.Cron, s.runWeeklyWrap)
	if err != nil {
		return err
	}

	// Start the cron scheduler
	s.cron.Start()

	log.Println("Scheduler started successfully")
	return nil
}

// RunOnce runs the weekly wrap job once (useful for testing/dry-run)
func (s *Scheduler) RunOnce() {
	s.runWeeklyWrap()
}

func (s *Scheduler) runWeeklyWrap() {
	log.Println("Running weekly wrap...")

	// Get current date and calculate week range
	now := time.Now()
	weekEnd := now
	weekStart := now.AddDate(0, 0, -7)

	log.Printf("Processing week from %s to %s", weekStart.Format("2006-01-02"), weekEnd.Format("2006-01-02"))

	// Get weekly data from YNAB
	data, err := s.ynabClient.GetWeeklyData(weekStart, weekEnd)
	if err != nil {
		log.Printf("Failed to get weekly data: %v", err)
		return
	}

	// Analyze the data
	topCategoriesLimit := s.config.Thresholds.TopCategoriesCount
	analysis, err := s.analyzer.AnalyzeWeeklyData(data, topCategoriesLimit)
	if err != nil {
		log.Printf("Failed to analyze data: %v", err)
		return
	}

	// Format the message
	message := s.formatMessage(analysis)

	if s.dryRun {
		separator := strings.Repeat("=", 80)
		log.Println("\n" + separator)
		log.Println("DRY RUN MODE - Output that would be sent to Telegram:")
		log.Println(separator)
		fmt.Println(message)
		log.Println(separator)
		log.Println("Weekly wrap dry-run completed successfully (not sent to Telegram)")
	} else if s.telegramBot != nil {
		// Send to Telegram
		err = s.telegramBot.SendWeeklyWrap(message)
		if err != nil {
			log.Printf("Failed to send Telegram message: %v", err)
			return
		}

		log.Println("Weekly wrap completed successfully")
	} else {
		log.Println("Telegram bot is not configured, skipping message send")
	}
}

// formatAmount formats a float amount, removing unnecessary decimals
func (s *Scheduler) formatAmount(amount float64) string {
	// Check if the amount is a whole number
	if amount == float64(int64(amount)) {
		return fmt.Sprintf("%.0f", amount)
	}
	// Otherwise show up to 2 decimals, but trim trailing zeros
	formatted := fmt.Sprintf("%.2f", amount)
	// Remove trailing zeros after decimal point
	formatted = strings.TrimRight(formatted, "0")
	formatted = strings.TrimRight(formatted, ".")
	return formatted
}

func (s *Scheduler) formatMessage(analysis *processor.AnalysisResult) string {
	// Format currency amounts (YNAB stores amounts in millicents)
	spent := float64(analysis.Overview.TotalSpent) / 1000
	spentStr := s.formatAmount(spent)

	// Create header with category count
	categoryCountText := "Spending Categories"
	if len(analysis.TopSpending) == 0 {
		categoryCountText = "No Spending Categories"
	} else if len(analysis.TopSpending) == 1 {
		categoryCountText = "1 Spending Category"
	} else {
		categoryCountText = fmt.Sprintf("%d Spending Categories", len(analysis.TopSpending))
	}

	message := fmt.Sprintf(
		"📊 **Weekly Financial Wrap - %s**\n\n"+
			"💰 **Total Spent**: $%s\n\n"+
			"🏆 **Top %s**\n",
		analysis.DateRange,
		spentStr,
		categoryCountText,
	)

	// Add top spending categories
	for _, category := range analysis.TopSpending {
		// Activity is stored as negative in YNAB, convert to positive
		catActivity := -float64(category.Activity) / 1000
		catBalance := float64(category.Balance) / 1000

		// Format amounts, removing unnecessary decimals
		activityStr := s.formatAmount(catActivity)
		balanceStr := s.formatAmount(catBalance)

		message += fmt.Sprintf("• **%s**: Activity: $%s  Remaining: $%s\n",
			category.Category, activityStr, balanceStr)
	}

	message += "\n⚠️ **Over Budget Categories**\n"

	// Add concerns with transaction details
	if len(analysis.Concerns) > 0 {
		for _, concern := range analysis.Concerns {
			spentAmount := float64(concern.Spent) / 1000
			balanceAmount := float64(concern.Balance) / 1000

			spentStr := s.formatAmount(spentAmount)
			balanceStr := s.formatAmount(balanceAmount)

			message += fmt.Sprintf("\n**%s**: Activity: $%s  Remaining: $%s\n",
				concern.Category, spentStr, balanceStr)

			// Add transaction details
			if len(concern.Transactions) > 0 {
				message += "Last 3 transactions:\n"
				for count, tx := range concern.Transactions {
					// YNAB stores spending as negative, convert to positive for display
					if count == 3 {
						break
					}
					txAmount := -float64(tx.Amount) / 1000
					txAmountStr := s.formatAmount(txAmount)
					date := ""
					if tx.Date != nil {
						date = tx.Date.Format("01-02")
					}
					memo := tx.Memo
					if memo == "" {
						memo = tx.PayeeName
					}
					message += fmt.Sprintf("  • %s: $%s - %s\n", date, txAmountStr, memo)
				}
			}
		}
	} else {
		message += "• No categories over budget - great job! 🎉\n"
	}

	return message
}
