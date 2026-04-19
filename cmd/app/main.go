package main

import (
	"flag"
	"log"
	"os"

	"github.com/sathyabhat/ynab-weekly-wrap/internal/config"
	"github.com/sathyabhat/ynab-weekly-wrap/internal/scheduler"
)

func main() {
	// Command-line flags
	dryRun := flag.Bool("dry-run", false, "Run once and print output to stdout without sending to Telegram")
	once := flag.Bool("once", false, "Run once and exit (for manual testing)")
	onceMonthly := flag.Bool("once-monthly", false, "Run monthly wrap once and exit")
	flag.Parse()

	log.Println("Starting YNAB Weekly Wrap...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration (skip Telegram validation in test modes)
	testMode := *dryRun || *once || *onceMonthly
	if err := config.ValidateConfig(cfg, testMode); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	log.Printf("Configuration loaded successfully")
	log.Printf("Budget ID: %s", cfg.YNAB.BudgetID)

	if *dryRun {
		log.Println("[DRY RUN MODE] Will print output to stdout instead of sending to publishers")
	}
	if *once {
		log.Println("[ONCE MODE] Will run once and exit")
	}
	if *onceMonthly {
		log.Println("[ONCE MONTHLY MODE] Will run monthly wrap once and exit")
	}

	// Initialize scheduler (skip Telegram only in dry-run mode)
	dryRunMode := *dryRun
	opts := []scheduler.SchedulerOption{scheduler.WithDryRun(dryRunMode)}
	if *dryRun {
		opts = append(opts, scheduler.WithSkipTelegram(true))
	}
	sched := scheduler.NewScheduler(cfg, opts...)

	// Monthly once — check before the weekly once/dry-run block
	if *onceMonthly {
		log.Println("Running monthly wrap once and exiting...")
		sched.RunMonthlyOnce()
		os.Exit(0)
	}

	// Run weekly once for testing if requested
	if *once || *dryRun {
		log.Println("Running once and exiting...")
		sched.RunOnce()
		os.Exit(0)
	}

	// Start the scheduler
	if err := sched.Start(); err != nil {
		log.Fatalf("Failed to start scheduler: %v", err)
	}

	// Keep the application running
	select {}
}
