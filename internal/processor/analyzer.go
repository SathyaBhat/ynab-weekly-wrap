package processor

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/sathyabhat/ynab-weekly-wrap/internal/ynab"
)

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (a *Analyzer) AnalyzeWeeklyData(data *ynab.WeeklyData, topCategoriesLimit int) (*AnalysisResult, error) {
	if data == nil {
		return nil, fmt.Errorf("weekly data is nil")
	}

	// Calculate spending by category
	categorySpending := a.calculateCategorySpending(data.Categories, data.Transactions)

	// Calculate budget health
	overview := a.calculateOverview(categorySpending)

	// Get top spending categories (0 = all, >0 = limit to N)
	topSpending := a.getTopSpendingCategories(categorySpending, topCategoriesLimit)

	// Identify budget wins
	wins := a.identifyWins(categorySpending)

	// Identify areas for attention (with transaction details)
	concerns := a.identifyConcernsWithTransactions(categorySpending)

	// Calculate ahead focus
	aheadFocus := a.calculateAheadFocus(categorySpending, data.WeekEnd)

	result := &AnalysisResult{
		Overview:    overview,
		TopSpending: topSpending,
		Wins:        wins,
		Concerns:    concerns,
		AheadFocus:  aheadFocus,
		DateRange:   data.WeekStart.Format("2006-01-02") + " to " + data.WeekEnd.Format("2006-01-02"),
	}

	return result, nil
}

func (a *Analyzer) calculateCategorySpending(categories []ynab.Category, transactions []ynab.Transaction) []CategorySpending {
	spendingMap := make(map[string]int64)
	txByCategory := make(map[string][]ynab.Transaction)

	// Sum transactions by category (only negative amounts = spending in YNAB)
	for _, tx := range transactions {
		if tx.Deleted || tx.CategoryID == nil || tx.Amount >= 0 {
			continue
		}
		// Use absolute value for spending
		spendingMap[tx.CategoryName] += -tx.Amount
		txByCategory[tx.CategoryName] = append(txByCategory[tx.CategoryName], tx)
	}

	// Create category spending list
	var categorySpendingList []CategorySpending
	for _, cat := range categories {
		if cat.Budgeted == 0 {
			continue
		}

		spend := spendingMap[cat.Name]
		percentage := float64(spend) / float64(cat.Budgeted) * 100

		categoryTxns := txByCategory[cat.Name]

		categorySpendingList = append(categorySpendingList, CategorySpending{
			Category:     cat,
			Spent:        spend,
			Budgeted:     cat.Budgeted,
			Balance:      cat.Balance, // Use YNAB's balance (remaining for the month)
			Percentage:   percentage,
			Transactions: categoryTxns,
		})
	}

	return categorySpendingList
}

func (a *Analyzer) calculateOverview(spending []CategorySpending) *Overview {
	totalSpent := int64(0)
	totalBudgeted := int64(0)
	totalBalance := int64(0)

	for _, cat := range spending {
		totalSpent += cat.Spent
		totalBudgeted += cat.Budgeted
		totalBalance += cat.Balance
	}

	healthPercentage := float64(0)
	if totalBudgeted > 0 {
		healthPercentage = float64(totalSpent) / float64(totalBudgeted) * 100
	}

	return &Overview{
		TotalSpent:       totalSpent,
		TotalBudgeted:    totalBudgeted,
		TotalBalance:     totalBalance,
		HealthPercentage: healthPercentage,
	}
}

func (a *Analyzer) identifyWins(spending []CategorySpending) []CategoryWin {
	var wins []CategoryWin

	// Sort by balance (descending) - categories with most money left
	sort.Slice(spending, func(i, j int) bool {
		return spending[i].Balance > spending[j].Balance
	})

	// Take top 3 wins (categories with highest remaining balance)
	for i := 0; i < 3 && i < len(spending); i++ {
		cat := spending[i]
		if cat.Balance > 0 { // Only include categories with positive balance
			wins = append(wins, CategoryWin{
				Category:   cat.Category.Name,
				Balance:    cat.Balance,
				Percentage: cat.Percentage,
			})
		}
	}

	return wins
}

func (a *Analyzer) identifyConcerns(spending []CategorySpending) []CategoryConcern {
	var concerns []CategoryConcern

	// Sort by balance (ascending) - most negative/lowest balance first
	sort.Slice(spending, func(i, j int) bool {
		return spending[i].Balance < spending[j].Balance
	})

	// Find categories that are over budget (negative balance)
	for _, cat := range spending {
		if cat.Balance < 0 {
			concerns = append(concerns, CategoryConcern{
				Category:   cat.Category.Name,
				Over:       -cat.Balance, // How much over budget
				Percentage: cat.Percentage,
			})
		}
	}

	return concerns
}

func (a *Analyzer) getTopSpendingCategories(spending []CategorySpending, limit int) []TopSpendingCategory {
	var topCategories []TopSpendingCategory

	// Filter out categories with zero spending
	var withSpending []CategorySpending
	for _, cat := range spending {
		if cat.Spent > 0 {
			withSpending = append(withSpending, cat)
		}
	}

	// Sort by spent amount (descending)
	sort.Slice(withSpending, func(i, j int) bool {
		return withSpending[i].Spent > withSpending[j].Spent
	})

	// If limit is 0, return all categories; otherwise limit to N
	actualLimit := limit
	if limit == 0 {
		actualLimit = len(withSpending)
	}

	// Take top N categories
	for i := 0; i < actualLimit && i < len(withSpending); i++ {
		cat := withSpending[i]
		topCategories = append(topCategories, TopSpendingCategory{
			Category:   cat.Category.Name,
			Spent:      cat.Spent,
			Budgeted:   cat.Budgeted,
			Balance:    cat.Balance,
			Percentage: cat.Percentage,
		})
	}

	return topCategories
}

func (a *Analyzer) identifyConcernsWithTransactions(spending []CategorySpending) []CategoryConcernWithTransactions {
	var concerns []CategoryConcernWithTransactions

	// Sort by balance (ascending - most negative first)
	sort.Slice(spending, func(i, j int) bool {
		return spending[i].Category.Balance < spending[j].Category.Balance
	})

	// Find categories that have negative balance (over budget)
	for _, cat := range spending {
		if cat.Category.Balance < 0 {
			// Calculate how much we're over the available balance
			overage := -cat.Category.Balance
			concerns = append(concerns, CategoryConcernWithTransactions{
				Category:     cat.Category.Name,
				Budgeted:     cat.Budgeted,
				Spent:        cat.Spent,
				Balance:      cat.Category.Balance,
				Over:         overage,
				Percentage:   cat.Percentage,
				Transactions: cat.Transactions,
			})
		}
	}

	return concerns
}

func (a *Analyzer) calculateAheadFocus(spending []CategorySpending, weekEnd time.Time) *AheadFocus {
	var highestRiskCategories []string
	var adjustments []string

	for _, cat := range spending {
		if cat.Percentage >= 75 && cat.Percentage < 100 {
			highestRiskCategories = append(highestRiskCategories, cat.Category.Name)
		}
		if cat.Percentage >= 100 {
			adjustments = append(adjustments, fmt.Sprintf("Consider reducing %s budget", cat.Category.Name))
		}
	}

	return &AheadFocus{
		Watch:       highestRiskCategories,
		Adjustments: adjustments,
		WeeksLeft:   int(math.Ceil(time.Until(weekEnd).Hours() / 24 / 7)),
	}
}
