package processor

import (
	"github.com/sathyabhat/ynab-weekly-wrap/internal/ynab"
)

type CategorySpending struct {
	Category     ynab.Category
	Spent        int64   // Weekly spending for this category
	Budgeted     int64   // Monthly budgeted amount
	Balance      int64   // Remaining balance for the month (from YNAB)
	Percentage   float64 // Percentage of monthly budget spent this week
	Transactions []ynab.Transaction
}

type AnalysisResult struct {
	Overview       *Overview
	TopSpending    []TopSpendingCategory
	Wins           []CategoryWin
	Concerns       []CategoryConcernWithTransactions
	AheadFocus     *AheadFocus
	DateRange      string
}

type Overview struct {
	TotalSpent       int64   // Total weekly spending across all categories
	TotalBudgeted    int64   // Total monthly budget across all categories
	TotalBalance     int64   // Total remaining balance for the month across all categories
	HealthPercentage float64 // Percentage of monthly budget used
}

type CategoryWin struct {
	Category   string
	Balance    int64   // Remaining balance for the month
	Percentage float64 // Percentage of monthly budget used
}

type CategoryConcern struct {
	Category   string
	Over       int64
	Percentage float64
}

type AheadFocus struct {
	Watch       []string
	Adjustments []string
	WeeksLeft   int
}

type TopSpendingCategory struct {
	Category   string
	Spent      int64   // Weekly spending for this category
	Budgeted   int64   // Monthly budgeted amount
	Balance    int64   // Remaining balance for the month
	Percentage float64 // Percentage of monthly budget spent this week
}

type CategoryConcernWithTransactions struct {
	Category     string
	Budgeted     int64
	Spent        int64
	Balance      int64
	Over         int64
	Percentage   float64
	Transactions []ynab.Transaction
}
