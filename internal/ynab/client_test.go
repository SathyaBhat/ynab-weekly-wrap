package ynab

import (
	"fmt"
	"testing"
	"time"
)

// mockFetcher implements dataFetcher for unit tests.
type mockFetcher struct {
	budget       *Budget
	categories   []Category
	transactions []Transaction
	budgetErr    error
	categoriesErr error
	transactionsErr error

	// captured args
	capturedBudgetID string
	capturedStart    time.Time
	capturedEnd      time.Time
}

func (m *mockFetcher) getBudget(budgetID string) (*Budget, error) {
	m.capturedBudgetID = budgetID
	return m.budget, m.budgetErr
}

func (m *mockFetcher) getCategories(budgetID string) ([]Category, error) {
	return m.categories, m.categoriesErr
}

func (m *mockFetcher) getTransactions(budgetID string, start, end time.Time) ([]Transaction, error) {
	m.capturedStart = start
	m.capturedEnd = end
	return m.transactions, m.transactionsErr
}

func newClientWithFetcher(budgetID string, f dataFetcher) *Client {
	c := &Client{fetcher: f}
	c.config.BudgetID = budgetID
	return c
}

func testBudget() *Budget {
	return &Budget{ID: "b1", Name: "Test Budget"}
}

func testCategories() []Category {
	return []Category{
		{ID: "c1", Name: "Groceries", Budgeted: 500_000, Balance: 300_000},
		{ID: "c2", Name: "Transport", Budgeted: 200_000, Balance: 50_000},
	}
}

// ── GetMonthlyData ────────────────────────────────────────────────────────────

func TestGetMonthlyData_DateBoundaries(t *testing.T) {
	cases := []struct {
		year      int
		month     int
		wantStart string
		wantEnd   string
	}{
		{2026, 1, "2026-01-01", "2026-01-31"},
		{2026, 2, "2026-02-01", "2026-02-28"}, // non-leap
		{2024, 2, "2024-02-01", "2024-02-29"}, // leap year
		{2025, 12, "2025-12-01", "2025-12-31"},
		{2026, 4, "2026-04-01", "2026-04-30"}, // 30-day month
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%d-%02d", tc.year, tc.month), func(t *testing.T) {
			mock := &mockFetcher{
				budget:     testBudget(),
				categories: testCategories(),
			}
			c := newClientWithFetcher("b1", mock)

			data, err := c.GetMonthlyData(tc.year, tc.month)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			gotStart := data.MonthStart.Format("2006-01-02")
			gotEnd := data.MonthEnd.Format("2006-01-02")

			if gotStart != tc.wantStart {
				t.Errorf("MonthStart: got %s, want %s", gotStart, tc.wantStart)
			}
			if gotEnd != tc.wantEnd {
				t.Errorf("MonthEnd: got %s, want %s", gotEnd, tc.wantEnd)
			}

			// The fetcher should receive the same boundaries
			if mock.capturedStart.Format("2006-01-02") != tc.wantStart {
				t.Errorf("capturedStart: got %s, want %s", mock.capturedStart.Format("2006-01-02"), tc.wantStart)
			}
			if mock.capturedEnd.Format("2006-01-02") != tc.wantEnd {
				t.Errorf("capturedEnd: got %s, want %s", mock.capturedEnd.Format("2006-01-02"), tc.wantEnd)
			}
		})
	}
}

func TestGetMonthlyData_ReturnsBudgetAndCategories(t *testing.T) {
	cats := testCategories()
	txDate := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	txs := []Transaction{
		{ID: "t1", Date: &txDate, Amount: -50_000, CategoryName: "Groceries"},
	}
	mock := &mockFetcher{
		budget:       testBudget(),
		categories:   cats,
		transactions: txs,
	}
	c := newClientWithFetcher("b1", mock)

	data, err := c.GetMonthlyData(2026, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data.Budget.ID != "b1" {
		t.Errorf("Budget.ID: got %s, want b1", data.Budget.ID)
	}
	if len(data.Categories) != len(cats) {
		t.Errorf("Categories count: got %d, want %d", len(data.Categories), len(cats))
	}
	if len(data.Transactions) != 1 {
		t.Errorf("Transactions count: got %d, want 1", len(data.Transactions))
	}
}

func TestGetMonthlyData_BudgetError(t *testing.T) {
	mock := &mockFetcher{budgetErr: fmt.Errorf("API down")}
	c := newClientWithFetcher("b1", mock)

	_, err := c.GetMonthlyData(2026, 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetMonthlyData_TransactionError(t *testing.T) {
	mock := &mockFetcher{
		budget:          testBudget(),
		categories:      testCategories(),
		transactionsErr: fmt.Errorf("timeout"),
	}
	c := newClientWithFetcher("b1", mock)

	_, err := c.GetMonthlyData(2026, 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ── GetWeeklyData ─────────────────────────────────────────────────────────────

func TestGetWeeklyData_PassesDateRange(t *testing.T) {
	weekStart := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
	weekEnd := time.Date(2026, 2, 23, 0, 0, 0, 0, time.UTC)

	mock := &mockFetcher{budget: testBudget(), categories: testCategories()}
	c := newClientWithFetcher("b1", mock)

	data, err := c.GetWeeklyData(weekStart, weekEnd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !data.WeekStart.Equal(weekStart) {
		t.Errorf("WeekStart: got %v, want %v", data.WeekStart, weekStart)
	}
	if !data.WeekEnd.Equal(weekEnd) {
		t.Errorf("WeekEnd: got %v, want %v", data.WeekEnd, weekEnd)
	}
}
