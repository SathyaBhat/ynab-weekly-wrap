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

type Client struct {
	config config.YNABConfig
	client ynab.ClientServicer
}

func NewClient(ynabConfig config.YNABConfig) *Client {
	return &Client{
		config: ynabConfig,
		client: ynab.NewClient(ynabConfig.APIToken),
	}
}

func (c *Client) GetWeeklyData(weekStart, weekEnd time.Time) (*WeeklyData, error) {
	log.Printf("Fetching weekly data from %s to %s", weekStart.Format("2006-01-02"), weekEnd.Format("2006-01-02"))

	budget, err := c.getBudget(c.config.BudgetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get budget: %w", err)
	}

	categories, err := c.getCategories(c.config.BudgetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	transactions, err := c.getTransactions(c.config.BudgetID, weekStart, weekEnd)
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

func (c *Client) getBudget(budgetID string) (*Budget, error) {
	budgetData, err := c.client.Budget().GetBudget(budgetID, nil)
	if err != nil {
		return nil, err
	}

	if budgetData == nil || budgetData.Budget == nil {
		return nil, fmt.Errorf("no budget data returned")
	}

	budget := &Budget{
		ID:           budgetData.Budget.ID,
		Name:         budgetData.Budget.Name,
		LastModified: budgetData.Budget.LastModifiedOn,
		Categories:   []Category{},
	}

	return budget, nil
}

func (c *Client) getCategories(budgetID string) ([]Category, error) {
	categoriesData, err := c.client.Category().GetCategories(budgetID, nil)
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
				Budgeted:      cat.Budgeted,
				Activity:      cat.Activity,
				Balance:       cat.Balance,
				TargetBalance: cat.Balance, // Use Balance as TargetBalance for now
			}
			categories = append(categories, category)
		}
	}

	return categories, nil
}

func (c *Client) getTransactions(budgetID string, weekStart, weekEnd time.Time) ([]Transaction, error) {
	// Convert time.Time to api.Date for the filter
	sinceDate, err := api.DateFromString(weekStart.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse since date: %w", err)
	}

	filter := &ynabtransaction.Filter{
		Since: &sinceDate,
	}

	transactionsData, err := c.client.Transaction().GetTransactions(budgetID, filter)
	if err != nil {
		return nil, err
	}

	if transactionsData == nil {
		return nil, fmt.Errorf("no transactions data returned")
	}

	var transactions []Transaction
	for _, t := range transactionsData {
		// Filter transactions within the date range (until date)
		txDate := t.Date.Time
		if txDate.Before(weekStart) || txDate.After(weekEnd) {
			continue
		}

		transaction := Transaction{
			ID:                t.ID,
			Date:              &t.Date.Time,
			Amount:            t.Amount,
			Memo:              ptrToString(t.Memo),
			Cleared:           string(t.Cleared),
			Approved:          t.Approved,
			FlagColor:         ptrFlagColorToString(t.FlagColor),
			AccountID:         t.AccountID,
			AccountName:       t.AccountName,
			PayeeID:           t.PayeeID,
			PayeeName:         ptrToString(t.PayeeName),
			CategoryID:        t.CategoryID,
			CategoryName:      ptrToString(t.CategoryName),
			TransferAccountID: t.TransferAccountID,
			ImportID:          t.ImportID,
			Deleted:           t.Deleted,
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

func ptrFlagColorToString(fc *ynabtransaction.FlagColor) string {
	if fc == nil {
		return ""
	}
	return string(*fc)
}
