package processor

import (
	"github.com/sathyabhat/ynab-weekly-wrap/internal/ynab"
)

type CategorySpending struct {
	Category     ynab.Category
	Spent        int64
	Budgeted     int64
	Remaining    int64
	Percentage   float64
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
	TotalSpent       int64
	TotalBudgeted    int64
	TotalRemaining   int64
	HealthPercentage float64
}

type CategoryWin struct {
	Category   string
	Saved      int64
	Percentage float64
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
	Spent      int64
	Budgeted   int64
	Percentage float64
}

type CategoryConcernWithTransactions struct {
	Category     string
	Over         int64
	Percentage   float64
	Transactions []ynab.Transaction
}
