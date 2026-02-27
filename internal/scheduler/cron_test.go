package scheduler

import (
	"strings"
	"testing"

	"github.com/sathyabhat/ynab-weekly-wrap/internal/processor"
)

func newTestScheduler() *Scheduler {
	return &Scheduler{dryRun: true}
}

// ── formatAmount ─────────────────────────────────────────────────────────────

func TestFormatAmount_WholeNumber(t *testing.T) {
	s := newTestScheduler()
	cases := []struct {
		in   float64
		want string
	}{
		{0, "0"},
		{100, "100"},
		{1234, "1234"},
		{-50, "-50"},
	}
	for _, tc := range cases {
		got := s.formatAmount(tc.in)
		if got != tc.want {
			t.Errorf("formatAmount(%v): got %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatAmount_Decimal(t *testing.T) {
	s := newTestScheduler()
	cases := []struct {
		in   float64
		want string
	}{
		{1.5, "1.5"},
		{1.05, "1.05"},
		{1.50, "1.5"},  // trailing zero trimmed
		{0.10, "0.1"},
	}
	for _, tc := range cases {
		got := s.formatAmount(tc.in)
		if got != tc.want {
			t.Errorf("formatAmount(%v): got %q, want %q", tc.in, got, tc.want)
		}
	}
}

// ── formatMonthlyMessage ─────────────────────────────────────────────────────

func makeAnalysis(dateRange string, totalSpent int64, topCategories []processor.TopSpendingCategory, concerns []processor.CategoryConcernWithTransactions) *processor.AnalysisResult {
	return &processor.AnalysisResult{
		DateRange: dateRange,
		Overview: &processor.Overview{
			TotalSpent:    totalSpent,
			TotalBudgeted: totalSpent * 2,
		},
		TopSpending: topCategories,
		Concerns:    concerns,
		Wins:        nil,
		AheadFocus:  nil,
	}
}

func TestFormatMonthlyMessage_Header(t *testing.T) {
	s := newTestScheduler()
	analysis := makeAnalysis("January 2026", 700_000, nil, nil)

	msg := s.formatMonthlyMessage(analysis)

	if !strings.Contains(msg, "Monthly Financial Wrap") {
		t.Error("monthly header missing 'Monthly Financial Wrap'")
	}
	if !strings.Contains(msg, "January 2026") {
		t.Error("monthly header missing date range 'January 2026'")
	}
}

func TestFormatMonthlyMessage_TotalSpent(t *testing.T) {
	s := newTestScheduler()
	// 700_000 millicents = $700
	analysis := makeAnalysis("January 2026", 700_000, nil, nil)

	msg := s.formatMonthlyMessage(analysis)

	if !strings.Contains(msg, "$700") {
		t.Errorf("message should contain '$700', got:\n%s", msg)
	}
}

func TestFormatMonthlyMessage_CategoryLabelIsMonthly(t *testing.T) {
	s := newTestScheduler()
	analysis := makeAnalysis("January 2026", 350_000, []processor.TopSpendingCategory{
		{Category: "Dining", Spent: 350_000, Budgeted: 300_000, Balance: -50_000},
	}, nil)

	msg := s.formatMonthlyMessage(analysis)

	if !strings.Contains(msg, "Last Month Spend:") {
		t.Errorf("category line should say 'Last Month Spend:', got:\n%s", msg)
	}
	// Must NOT say "Last Week Spend:" in a monthly message
	if strings.Contains(msg, "Last Week Spend:") {
		t.Errorf("monthly message should not contain 'Last Week Spend:', got:\n%s", msg)
	}
}

func TestFormatMonthlyMessage_OverBudgetSection(t *testing.T) {
	s := newTestScheduler()
	concerns := []processor.CategoryConcernWithTransactions{
		{
			Category: "Dining",
			Spent:    350_000,
			Budgeted: 300_000,
			Balance:  -50_000,
			Over:     50_000,
		},
	}
	analysis := makeAnalysis("January 2026", 350_000, nil, concerns)

	msg := s.formatMonthlyMessage(analysis)

	if !strings.Contains(msg, "Over Budget") {
		t.Error("message missing 'Over Budget' section")
	}
	if !strings.Contains(msg, "Dining") {
		t.Error("message missing over-budget category 'Dining'")
	}
	if !strings.Contains(msg, "Last Month Spend:") {
		t.Error("over-budget line should say 'Last Month Spend:'")
	}
}

func TestFormatMonthlyMessage_NoConcerns(t *testing.T) {
	s := newTestScheduler()
	analysis := makeAnalysis("January 2026", 200_000, nil, nil)

	msg := s.formatMonthlyMessage(analysis)

	if !strings.Contains(msg, "No categories over budget") {
		t.Errorf("expected 'No categories over budget' when concerns empty, got:\n%s", msg)
	}
}

func TestFormatMonthlyMessage_CategoryCount_None(t *testing.T) {
	s := newTestScheduler()
	analysis := makeAnalysis("January 2026", 0, nil, nil)
	msg := s.formatMonthlyMessage(analysis)
	if !strings.Contains(msg, "No Spending Categories") {
		t.Errorf("expected 'No Spending Categories', got:\n%s", msg)
	}
}

func TestFormatMonthlyMessage_CategoryCount_One(t *testing.T) {
	s := newTestScheduler()
	analysis := makeAnalysis("January 2026", 100_000, []processor.TopSpendingCategory{
		{Category: "Groceries", Spent: 100_000, Budgeted: 500_000, Balance: 400_000},
	}, nil)
	msg := s.formatMonthlyMessage(analysis)
	if !strings.Contains(msg, "1 Spending Category") {
		t.Errorf("expected '1 Spending Category', got:\n%s", msg)
	}
}

func TestFormatMonthlyMessage_CategoryCount_Multiple(t *testing.T) {
	s := newTestScheduler()
	analysis := makeAnalysis("January 2026", 300_000, []processor.TopSpendingCategory{
		{Category: "Groceries", Spent: 200_000, Budgeted: 500_000, Balance: 300_000},
		{Category: "Transport", Spent: 100_000, Budgeted: 200_000, Balance: 100_000},
	}, nil)
	msg := s.formatMonthlyMessage(analysis)
	if !strings.Contains(msg, "2 Spending Categories") {
		t.Errorf("expected '2 Spending Categories', got:\n%s", msg)
	}
}

// ── formatMessage (weekly) ────────────────────────────────────────────────────

func TestFormatMessage_Header(t *testing.T) {
	s := newTestScheduler()
	analysis := makeAnalysis("2026-01-19 to 2026-01-26", 200_000, nil, nil)

	msg := s.formatMessage(analysis)

	if !strings.Contains(msg, "Weekly Financial Wrap") {
		t.Error("weekly header missing 'Weekly Financial Wrap'")
	}
}

func TestFormatMessage_CategoryLabelIsWeekly(t *testing.T) {
	s := newTestScheduler()
	analysis := makeAnalysis("2026-01-19 to 2026-01-26", 200_000, []processor.TopSpendingCategory{
		{Category: "Groceries", Spent: 200_000, Budgeted: 500_000, Balance: 300_000},
	}, nil)

	msg := s.formatMessage(analysis)

	if !strings.Contains(msg, "Last Week Spend:") {
		t.Errorf("category line should say 'Last Week Spend:', got:\n%s", msg)
	}
}

func TestFormatMessage_NoConcerns(t *testing.T) {
	s := newTestScheduler()
	analysis := makeAnalysis("2026-01-19 to 2026-01-26", 200_000, nil, nil)
	msg := s.formatMessage(analysis)
	if !strings.Contains(msg, "No categories over budget") {
		t.Errorf("expected 'No categories over budget', got:\n%s", msg)
	}
}
