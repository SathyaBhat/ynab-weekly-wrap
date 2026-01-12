package ynab

import (
	"time"
)

type Budget struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	LastModified *time.Time `json:"last_modified"`
	Categories   []Category `json:"categories"`
}

type Category struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	CategoryGroupID string        `json:"category_group_id"`
	CategoryGroup   CategoryGroup `json:"category_group"`
	Budgeted        int64         `json:"budgeted"`
	Activity        int64         `json:"activity"`
	Balance         int64         `json:"balance"`
	TargetBalance   int64         `json:"target_balance"`
}

type CategoryGroup struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Hidden  bool   `json:"hidden"`
	Deleted bool   `json:"deleted"`
}

type Transaction struct {
	ID                string     `json:"id"`
	Date              *time.Time `json:"date"`
	Amount            int64      `json:"amount"`
	Memo              string     `json:"memo"`
	Cleared           string     `json:"cleared"`
	Approved          bool       `json:"approved"`
	FlagColor         string     `json:"flag_color"`
	AccountID         string     `json:"account_id"`
	AccountName       string     `json:"account_name"`
	PayeeID           *string    `json:"payee_id"`
	PayeeName         string     `json:"payee_name"`
	CategoryID        *string    `json:"category_id"`
	CategoryName      string     `json:"category_name"`
	TransferAccountID *string    `json:"transfer_account_id"`
	ImportID          *string    `json:"import_id"`
	Deleted           bool       `json:"deleted"`
}

type WeeklyData struct {
	Budget       *Budget
	Categories   []Category
	Transactions []Transaction
	WeekStart    time.Time
	WeekEnd      time.Time
}

type CategorySpending struct {
	Category     Category
	Spent        int64
	Budgeted     int64
	Remaining    int64
	Percentage   float64
	Transactions []Transaction
}
