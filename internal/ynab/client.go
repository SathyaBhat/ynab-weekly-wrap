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

// dataFetcher abstracts the YNAB API calls used by Client,
// allowing tests to inject a mock without a real network connection.
type dataFetcher interface {
	getBudget(budgetID string) (*Budget, error)
	getCategories(budgetID string) ([]Category, error)
	getTransactions(budgetID string, start, end time.Time) ([]Transaction, error)
	getMonthCategories(budgetID string, year, month int) ([]Category, error)
	getMonthCategoryActivity(budgetID string, year, month int) (map[string]int64, error)
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

	log.Printf("Fetching monthly data for %s", monthStart.Format("January 2006"))

	budget, err := c.fetcher.getBudget(c.config.BudgetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get budget: %w", err)
	}

	categories, err := c.fetcher.getMonthCategories(c.config.BudgetID, year, month)
	if err != nil {
		return nil, fmt.Errorf("failed to get monthly categories: %w", err)
	}

	log.Printf("Retrieved %d categories", len(categories))

	return &MonthlyData{
		Budget:     budget,
		Categories: categories,
		MonthStart: monthStart,
		MonthEnd:   monthEnd,
	}, nil
}

func (c *Client) GetPrevMonthCategorySpend(year, month int) (map[string]int64, error) {
	log.Printf("Fetching category activity for %04d-%02d", year, month)
	return c.fetcher.getMonthCategoryActivity(c.config.BudgetID, year, month)
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

func (a *apiClient) getMonthCategories(budgetID string, year, month int) ([]Category, error) {
	monthStr := fmt.Sprintf("%04d-%02d-01", year, month)
	date, err := api.DateFromString(monthStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse month date: %w", err)
	}

	monthData, err := a.client.Month().GetMonth(budgetID, date)
	if err != nil {
		return nil, err
	}

	if monthData == nil {
		return nil, fmt.Errorf("no month data returned")
	}

	var categories []Category
	for _, cat := range monthData.Categories {
		if cat == nil {
			continue
		}
		categories = append(categories, Category{
			ID:       cat.ID,
			Name:     cat.Name,
			Budgeted: cat.Budgeted,
			Activity: cat.Activity,
			Balance:  cat.Balance,
		})
	}
	return categories, nil
}

func (a *apiClient) getMonthCategoryActivity(budgetID string, year, month int) (map[string]int64, error) {
	monthStr := fmt.Sprintf("%04d-%02d-01", year, month)
	date, err := api.DateFromString(monthStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse month date: %w", err)
	}

	monthData, err := a.client.Month().GetMonth(budgetID, date)
	if err != nil {
		return nil, err
	}

	if monthData == nil {
		return nil, fmt.Errorf("no month data returned")
	}

	result := make(map[string]int64)
	for _, cat := range monthData.Categories {
		if cat == nil || cat.Activity >= 0 {
			continue
		}
		result[cat.Name] = -cat.Activity
	}
	return result, nil
}

// Helper functions to convert between types
func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

