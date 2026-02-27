package processor

import (
	"testing"
	"time"

	"github.com/sathyabhat/ynab-weekly-wrap/internal/ynab"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func makeDate(year, month, day int) *time.Time {
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	return &t
}

func makeCategory(id, name string, budgeted, balance int64) ynab.Category {
	return ynab.Category{
		ID:       id,
		Name:     name,
		Budgeted: budgeted,
		Balance:  balance,
	}
}

func makeTx(id string, date *time.Time, amount int64, categoryName string) ynab.Transaction {
	catID := categoryName // non-nil pointer so the analyzer doesn't skip it
	return ynab.Transaction{
		ID:           id,
		Date:         date,
		Amount:       amount,
		CategoryName: categoryName,
		CategoryID:   &catID,
	}
}

func baseMonthlyData() *ynab.MonthlyData {
	return &ynab.MonthlyData{
		Budget: &ynab.Budget{ID: "b1", Name: "Test Budget"},
		Categories: []ynab.Category{
			makeCategory("c1", "Groceries", 500_000, 300_000),
			makeCategory("c2", "Transport", 200_000, 50_000),
			makeCategory("c3", "Dining", 300_000, -50_000), // over budget
		},
		Transactions: []ynab.Transaction{
			makeTx("t1", makeDate(2026, 1, 5), -200_000, "Groceries"),
			makeTx("t2", makeDate(2026, 1, 10), -150_000, "Transport"),
			makeTx("t3", makeDate(2026, 1, 20), -350_000, "Dining"),
		},
		MonthStart: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		MonthEnd:   time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC),
	}
}

func baseWeeklyData() *ynab.WeeklyData {
	return &ynab.WeeklyData{
		Budget:       &ynab.Budget{ID: "b1", Name: "Test Budget"},
		Categories:   baseMonthlyData().Categories,
		Transactions: baseMonthlyData().Transactions,
		WeekStart:    time.Date(2026, 1, 19, 0, 0, 0, 0, time.UTC),
		WeekEnd:      time.Date(2026, 1, 26, 0, 0, 0, 0, time.UTC),
	}
}

// ── AnalyzeMonthlyData ────────────────────────────────────────────────────────

func TestAnalyzeMonthlyData_NilInput(t *testing.T) {
	a := NewAnalyzer()
	_, err := a.AnalyzeMonthlyData(nil, 5)
	if err == nil {
		t.Fatal("expected error for nil input, got nil")
	}
}

func TestAnalyzeMonthlyData_DateRangeFormat(t *testing.T) {
	a := NewAnalyzer()
	result, err := a.AnalyzeMonthlyData(baseMonthlyData(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "January 2026"
	if result.DateRange != want {
		t.Errorf("DateRange: got %q, want %q", result.DateRange, want)
	}
}

func TestAnalyzeMonthlyData_AheadFocusIsNil(t *testing.T) {
	a := NewAnalyzer()
	result, err := a.AnalyzeMonthlyData(baseMonthlyData(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.AheadFocus != nil {
		t.Errorf("AheadFocus should be nil for monthly analysis, got %+v", result.AheadFocus)
	}
}

func TestAnalyzeMonthlyData_TotalSpent(t *testing.T) {
	a := NewAnalyzer()
	result, err := a.AnalyzeMonthlyData(baseMonthlyData(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Groceries: 200_000 + Transport: 150_000 + Dining: 350_000 = 700_000
	wantSpent := int64(700_000)
	if result.Overview.TotalSpent != wantSpent {
		t.Errorf("TotalSpent: got %d, want %d", result.Overview.TotalSpent, wantSpent)
	}
}

func TestAnalyzeMonthlyData_OverBudgetConcerns(t *testing.T) {
	a := NewAnalyzer()
	result, err := a.AnalyzeMonthlyData(baseMonthlyData(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Concerns) != 1 {
		t.Fatalf("Concerns count: got %d, want 1", len(result.Concerns))
	}
	if result.Concerns[0].Category != "Dining" {
		t.Errorf("Concerns[0].Category: got %s, want Dining", result.Concerns[0].Category)
	}
	if result.Concerns[0].Over != 50_000 {
		t.Errorf("Concerns[0].Over: got %d, want 50000", result.Concerns[0].Over)
	}
}

func TestAnalyzeMonthlyData_TopSpendingLimit(t *testing.T) {
	a := NewAnalyzer()

	// limit=2 — only top 2 categories returned
	result, err := a.AnalyzeMonthlyData(baseMonthlyData(), 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.TopSpending) != 2 {
		t.Errorf("TopSpending count with limit=2: got %d, want 2", len(result.TopSpending))
	}
	// Top spender should be Dining (350_000)
	if result.TopSpending[0].Category != "Dining" {
		t.Errorf("TopSpending[0]: got %s, want Dining", result.TopSpending[0].Category)
	}
}

func TestAnalyzeMonthlyData_TopSpendingLimitZeroReturnsAll(t *testing.T) {
	a := NewAnalyzer()
	result, err := a.AnalyzeMonthlyData(baseMonthlyData(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.TopSpending) != 3 {
		t.Errorf("TopSpending count with limit=0: got %d, want 3", len(result.TopSpending))
	}
}

func TestAnalyzeMonthlyData_NoTransactions(t *testing.T) {
	data := baseMonthlyData()
	data.Transactions = nil

	a := NewAnalyzer()
	result, err := a.AnalyzeMonthlyData(data, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Overview.TotalSpent != 0 {
		t.Errorf("TotalSpent with no transactions: got %d, want 0", result.Overview.TotalSpent)
	}
	if len(result.TopSpending) != 0 {
		t.Errorf("TopSpending with no transactions: got %d, want 0", len(result.TopSpending))
	}
}

func TestAnalyzeMonthlyData_DeletedTransactionsIgnored(t *testing.T) {
	data := baseMonthlyData()
	data.Transactions = []ynab.Transaction{
		makeTx("t1", makeDate(2026, 1, 5), -200_000, "Groceries"),
		{ID: "t2", Date: makeDate(2026, 1, 10), Amount: -999_000, CategoryName: "Groceries", Deleted: true},
	}

	a := NewAnalyzer()
	result, err := a.AnalyzeMonthlyData(data, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Overview.TotalSpent != 200_000 {
		t.Errorf("TotalSpent: got %d, want 200000 (deleted tx ignored)", result.Overview.TotalSpent)
	}
}

func TestAnalyzeMonthlyData_PositiveAmountsIgnored(t *testing.T) {
	data := baseMonthlyData()
	data.Transactions = []ynab.Transaction{
		makeTx("t1", makeDate(2026, 1, 5), -100_000, "Groceries"), // spend
		makeTx("t2", makeDate(2026, 1, 6), 50_000, "Groceries"),   // refund/income — ignored
	}

	a := NewAnalyzer()
	result, err := a.AnalyzeMonthlyData(data, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Overview.TotalSpent != 100_000 {
		t.Errorf("TotalSpent: got %d, want 100000", result.Overview.TotalSpent)
	}
}

func TestAnalyzeMonthlyData_NoBudgetedCategoriesSkipped(t *testing.T) {
	data := &ynab.MonthlyData{
		Budget: &ynab.Budget{ID: "b1"},
		Categories: []ynab.Category{
			makeCategory("c1", "Groceries", 500_000, 300_000),
			makeCategory("c2", "Uncategorized", 0, -100_000), // Budgeted=0, skip
		},
		Transactions: []ynab.Transaction{
			makeTx("t1", makeDate(2026, 1, 5), -200_000, "Groceries"),
		},
		MonthStart: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		MonthEnd:   time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC),
	}

	a := NewAnalyzer()
	result, err := a.AnalyzeMonthlyData(data, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.TopSpending) != 1 {
		t.Errorf("TopSpending count: got %d, want 1 (zero-budgeted categories skipped)", len(result.TopSpending))
	}
}

// ── AnalyzeWeeklyData ─────────────────────────────────────────────────────────

func TestAnalyzeWeeklyData_NilInput(t *testing.T) {
	a := NewAnalyzer()
	_, err := a.AnalyzeWeeklyData(nil, 5)
	if err == nil {
		t.Fatal("expected error for nil input, got nil")
	}
}

func TestAnalyzeWeeklyData_DateRangeFormat(t *testing.T) {
	a := NewAnalyzer()
	result, err := a.AnalyzeWeeklyData(baseWeeklyData(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "2026-01-19 to 2026-01-26"
	if result.DateRange != want {
		t.Errorf("DateRange: got %q, want %q", result.DateRange, want)
	}
}

func TestAnalyzeWeeklyData_AheadFocusNotNil(t *testing.T) {
	a := NewAnalyzer()
	result, err := a.AnalyzeWeeklyData(baseWeeklyData(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AheadFocus == nil {
		t.Error("AheadFocus should not be nil for weekly analysis")
	}
}

func TestAnalyzeWeeklyData_OverBudgetConcern(t *testing.T) {
	a := NewAnalyzer()
	result, err := a.AnalyzeWeeklyData(baseWeeklyData(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Concerns) != 1 {
		t.Fatalf("Concerns count: got %d, want 1", len(result.Concerns))
	}
	if result.Concerns[0].Category != "Dining" {
		t.Errorf("Concerns[0].Category: got %s, want Dining", result.Concerns[0].Category)
	}
}

// ── HealthPercentage ─────────────────────────────────────────────────────────

func TestAnalyzeMonthlyData_HealthPercentage(t *testing.T) {
	data := &ynab.MonthlyData{
		Budget: &ynab.Budget{},
		Categories: []ynab.Category{
			makeCategory("c1", "Groceries", 400_000, 200_000),
		},
		Transactions: []ynab.Transaction{
			makeTx("t1", makeDate(2026, 1, 1), -200_000, "Groceries"),
		},
		MonthStart: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		MonthEnd:   time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC),
	}

	a := NewAnalyzer()
	result, err := a.AnalyzeMonthlyData(data, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 200_000 / 400_000 * 100 = 50%
	if result.Overview.HealthPercentage != 50.0 {
		t.Errorf("HealthPercentage: got %.2f, want 50.00", result.Overview.HealthPercentage)
	}
}

// ── Wins ──────────────────────────────────────────────────────────────────────

func TestAnalyzeMonthlyData_WinsAreCategoriesWithPositiveBalance(t *testing.T) {
	a := NewAnalyzer()
	result, err := a.AnalyzeMonthlyData(baseMonthlyData(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Dining has negative balance → should not be a win
	for _, w := range result.Wins {
		if w.Category == "Dining" {
			t.Error("Dining (negative balance) should not appear in wins")
		}
	}
}
