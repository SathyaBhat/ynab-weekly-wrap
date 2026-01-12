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
	cron            *cron.Cron
	config          *config.Config
	ynabClient      *ynab.Client
	telegramBot     *telegram.Bot
	analyzer        *processor.Analyzer
	dryRun          bool
	skipTelegram    bool
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
	var location *time.Location
	
	// Try to load the specified timezone
	loc, err := time.LoadLocation(cfg.Schedule.Timezone)
	if err == nil {
		location = loc
	} else {
		// Fallback: try to use system local timezone
		loc, err := time.LoadLocation("Local")
		if err == nil {
			log.Printf("Warning: Could not load timezone '%s', using system local timezone", cfg.Schedule.Timezone)
			location = loc
		} else {
			// Final fallback: use UTC
			log.Printf("Warning: Could not load timezone '%s' or system local timezone, using UTC", cfg.Schedule.Timezone)
			location = time.UTC
		}
	}

	cronScheduler := cron.New(cron.WithLocation(location))

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
	log.Printf("Starting scheduler with cron expression: %s in timezone: %s", s.config.Schedule.Cron, s.config.Schedule.Timezone)

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
	analysis, err := s.analyzer.AnalyzeWeeklyData(data)
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

func (s *Scheduler) formatMessage(analysis *processor.AnalysisResult) string {
	// Format currency amounts (YNAB stores amounts in millicents)
	spent := float64(analysis.Overview.TotalSpent) / 1000

	message := fmt.Sprintf(
		"📊 **Weekly Financial Wrap - %s**\n\n"+
			"💰 **Total Spent**: $%.2f\n\n"+
			"🏆 **Top 5 Spending Categories**\n",
		analysis.DateRange,
		spent,
	)

	// Add top 5 spending categories
	for i, category := range analysis.TopSpending {
		if i >= 5 {
			break
		}
		catSpent := float64(category.Spent) / 1000
		catBudgeted := float64(category.Budgeted) / 1000
		message += fmt.Sprintf("• **%s**: $%.2f / $%.2f budgeted (%.1f%%)\n",
			category.Category, catSpent, catBudgeted, category.Percentage)
	}

	message += "\n⚠️ **Over Budget Categories**\n"

	// Add concerns with transaction details
	if len(analysis.Concerns) > 0 {
		for _, concern := range analysis.Concerns {
			overAmount := float64(concern.Over) / 1000
			message += fmt.Sprintf("\n**%s**: $%.2f over budget (%.1f%% of budget)\n",
				concern.Category, overAmount, concern.Percentage)
			
			// Add transaction details
			if len(concern.Transactions) > 0 {
				message += "Transactions:\n"
				for _, tx := range concern.Transactions {
					// YNAB stores spending as negative, convert to positive for display
					txAmount := -float64(tx.Amount) / 1000
					date := ""
					if tx.Date != nil {
						date = tx.Date.Format("01-02")
					}
					memo := tx.Memo
					if memo == "" {
						memo = tx.PayeeName
					}
					message += fmt.Sprintf("  • %s: $%.2f - %s\n", date, txAmount, memo)
				}
			}
		}
	} else {
		message += "• No categories over budget - great job! 🎉\n"
	}

	return message
}
