package ynab

import (
	"fmt"
	"log"
	"time"

	"github.com/brunomvsouza/ynab.go"
	"github.com/brunomvsouza/ynab.go/api"
	ynabtransaction "github.com/brunomvsouza/ynab.go/api/transaction"
	"github.com/sathyabhat/ynab-weekly-wrap/internal/config"
)

// dataFetcher abstracts the three YNAB API calls used by Client,
// allowing tests to inject a mock without a real network connection.
type dataFetcher interface {
	getBudget(budgetID string) (*Budget, error)
	getCategories(budgetID string) ([]Category, error)
	getTransactions(budgetID string, start, end time.Time) ([]Transaction, error)
}

type Client struct {
	config  config.YNABConfig
	fetcher dataFetcher
}

func NewClient(ynabConfig config.YNABConfig) *Client {
	return &Client{
		config:  ynabConfig,
		fetcher: &apiClient{client: ynab.NewClient(ynabConfig.APIToken)},
	}
}

func (c *Client) GetWeeklyData(weekStart, weekEnd time.Time) (*WeeklyData, error) {
	log.Printf("Fetching weekly data from %s to %s", weekStart.Format("2006-01-02"), weekEnd.Format("2006-01-02"))

	budget, err := c.fetcher.getBudget(c.config.BudgetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get budget: %w", err)
	}

	categories, err := c.fetcher.getCategories(c.config.BudgetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	transactions, err := c.fetcher.getTransactions(c.config.BudgetID, weekStart, weekEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	log.Printf("Retrieved %d categories and %d transactions", len(categories), len(transactions))

	return &WeeklyData{
		Budget:       budget,
		Categories:   categories,
		Transactions: transactions,
		WeekStart:    weekStart,
		WeekEnd:      weekEnd,
	}, nil
}

func (c *Client) GetMonthlyData(year, month int) (*MonthlyData, error) {
	monthStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, -1)

	log.Printf("Fetching monthly data for %s to %s", monthStart.Format("2006-01-02"), monthEnd.Format("2006-01-02"))

	budget, err := c.fetcher.getBudget(c.config.BudgetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get budget: %w", err)
	}

	categories, err := c.fetcher.getCategories(c.config.BudgetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	transactions, err := c.fetcher.getTransactions(c.config.BudgetID, monthStart, monthEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	log.Printf("Retrieved %d categories and %d transactions", len(categories), len(transactions))

	return &MonthlyData{
		Budget:       budget,
		Categories:   categories,
		Transactions: transactions,
		MonthStart:   monthStart,
		MonthEnd:     monthEnd,
	}, nil
}

// apiClient is the real implementation of dataFetcher, delegating to the YNAB library.
type apiClient struct {
	client ynab.ClientServicer
}

func (a *apiClient) getBudget(budgetID string) (*Budget, error) {
	budgetData, err := a.client.Budget().GetBudget(budgetID, nil)
	if err != nil {
		return nil, err
	}

	if budgetData == nil || budgetData.Budget == nil {
		return nil, fmt.Errorf("no budget data returned")
	}

	budget := &Budget{
		ID:   budgetData.Budget.ID,
		Name: budgetData.Budget.Name,
	}

	return budget, nil
}

func (a *apiClient) getCategories(budgetID string) ([]Category, error) {
	categoriesData, err := a.client.Category().GetCategories(budgetID, nil)
	if err != nil {
		return nil, err
	}

	if categoriesData == nil {
		return nil, fmt.Errorf("no categories data returned")
	}

	var categories []Category
	for _, group := range categoriesData.GroupWithCategories {
		for _, cat := range group.Categories {
			category := Category{
				ID:              cat.ID,
				Name:            cat.Name,
				CategoryGroupID: cat.CategoryGroupID,
				CategoryGroup: CategoryGroup{
					ID:      group.ID,
					Name:    group.Name,
					Hidden:  group.Hidden,
					Deleted: group.Deleted,
				},
				Budgeted: cat.Budgeted,
				Balance:  cat.Balance,
			}
			categories = append(categories, category)
		}
	}

	return categories, nil
}

func (a *apiClient) getTransactions(budgetID string, start, end time.Time) ([]Transaction, error) {
	sinceDate, err := api.DateFromString(start.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse since date: %w", err)
	}

	filter := &ynabtransaction.Filter{
		Since: &sinceDate,
	}

	transactionsData, err := a.client.Transaction().GetTransactions(budgetID, filter)
	if err != nil {
		return nil, err
	}

	if transactionsData == nil {
		return nil, fmt.Errorf("no transactions data returned")
	}

	var transactions []Transaction
	for _, t := range transactionsData {
		txDate := t.Date.Time
		if txDate.Before(start) || txDate.After(end) {
			continue
		}

		transaction := Transaction{
			ID:           t.ID,
			Date:         &t.Date.Time,
			Amount:       t.Amount,
			Memo:         ptrToString(t.Memo),
			AccountID:    t.AccountID,
			AccountName:  t.AccountName,
			PayeeID:      t.PayeeID,
			PayeeName:    ptrToString(t.PayeeName),
			CategoryID:   t.CategoryID,
			CategoryName: ptrToString(t.CategoryName),
			Deleted:      t.Deleted,
		}
		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

// Helper functions to convert between types
func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

