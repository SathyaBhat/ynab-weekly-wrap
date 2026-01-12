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

func (a *Analyzer) AnalyzeWeeklyData(data *ynab.WeeklyData) (*AnalysisResult, error) {
	if data == nil {
		return nil, fmt.Errorf("weekly data is nil")
	}

	// Calculate spending by category
	categorySpending := a.calculateCategorySpending(data.Categories, data.Transactions)

	// Calculate budget health
	overview := a.calculateOverview(categorySpending)

	// Get top 5 spending categories
	topSpending := a.getTopSpendingCategories(categorySpending, 5)

	// Identify budget wins
	wins := a.identifyWins(categorySpending)

	// Identify areas for attention (with transaction details)
	concerns := a.identifyConcernsWithTransactions(categorySpending)

	// Calculate ahead focus
	aheadFocus := a.calculateAheadFocus(categorySpending, data.WeekEnd)

	result := &AnalysisResult{
		Overview:       overview,
		TopSpending:    topSpending,
		Wins:           wins,
		Concerns:       concerns,
		AheadFocus:     aheadFocus,
		DateRange:      data.WeekStart.Format("2006-01-02") + " to " + data.WeekEnd.Format("2006-01-02"),
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
		remaining := cat.Budgeted - spend
		percentage := float64(spend) / float64(cat.Budgeted) * 100

		categoryTxns := txByCategory[cat.Name]

		categorySpendingList = append(categorySpendingList, CategorySpending{
			Category:     cat,
			Spent:        spend,
			Budgeted:     cat.Budgeted,
			Remaining:    remaining,
			Percentage:   percentage,
			Transactions: categoryTxns,
		})
	}

	return categorySpendingList
}

func (a *Analyzer) calculateOverview(spending []CategorySpending) *Overview {
	totalSpent := int64(0)
	totalBudgeted := int64(0)
	totalRemaining := int64(0)

	for _, cat := range spending {
		totalSpent += cat.Spent
		totalBudgeted += cat.Budgeted
		totalRemaining += cat.Remaining
	}

	healthPercentage := float64(0)
	if totalBudgeted > 0 {
		healthPercentage = float64(totalSpent) / float64(totalBudgeted) * 100
	}

	return &Overview{
		TotalSpent:       totalSpent,
		TotalBudgeted:    totalBudgeted,
		TotalRemaining:   totalRemaining,
		HealthPercentage: healthPercentage,
	}
}

func (a *Analyzer) identifyWins(spending []CategorySpending) []CategoryWin {
	var wins []CategoryWin

	// Sort by remaining amount (descending)
	sort.Slice(spending, func(i, j int) bool {
		return spending[i].Remaining > spending[j].Remaining
	})

	// Take top 3 wins
	for i := 0; i < 3 && i < len(spending); i++ {
		cat := spending[i]
		wins = append(wins, CategoryWin{
			Category:   cat.Category.Name,
			Saved:      cat.Remaining,
			Percentage: cat.Percentage,
		})
	}

	return wins
}

func (a *Analyzer) identifyConcerns(spending []CategorySpending) []CategoryConcern {
	var concerns []CategoryConcern

	// Sort by overspending (descending)
	sort.Slice(spending, func(i, j int) bool {
		return spending[i].Remaining < spending[j].Remaining
	})

	// Find categories that are over budget or near limit
	for _, cat := range spending {
		if cat.Remaining < 0 {
			concerns = append(concerns, CategoryConcern{
				Category:   cat.Category.Name,
				Over:       -cat.Remaining,
				Percentage: cat.Percentage,
			})
		}
	}

	return concerns
}

func (a *Analyzer) getTopSpendingCategories(spending []CategorySpending, limit int) []TopSpendingCategory {
	var topCategories []TopSpendingCategory

	// Sort by spent amount (descending)
	sort.Slice(spending, func(i, j int) bool {
		return spending[i].Spent > spending[j].Spent
	})

	// Take top N categories
	for i := 0; i < limit && i < len(spending); i++ {
		cat := spending[i]
		topCategories = append(topCategories, TopSpendingCategory{
			Category:  cat.Category.Name,
			Spent:     cat.Spent,
			Budgeted:  cat.Budgeted,
			Percentage: cat.Percentage,
		})
	}

	return topCategories
}

func (a *Analyzer) identifyConcernsWithTransactions(spending []CategorySpending) []CategoryConcernWithTransactions {
	var concerns []CategoryConcernWithTransactions

	// Sort by overspending (descending)
	sort.Slice(spending, func(i, j int) bool {
		return spending[i].Remaining < spending[j].Remaining
	})

	// Find categories that are over budget
	for _, cat := range spending {
		if cat.Remaining < 0 {
			concerns = append(concerns, CategoryConcernWithTransactions{
				Category:     cat.Category.Name,
				Over:         -cat.Remaining,
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
