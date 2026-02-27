package ynab

import (
	"time"
)

type Budget struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Category struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	CategoryGroupID string        `json:"category_group_id"`
	CategoryGroup   CategoryGroup `json:"category_group"`
	Budgeted        int64         `json:"budgeted"`
	Balance         int64         `json:"balance"`
}

type CategoryGroup struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Hidden  bool   `json:"hidden"`
	Deleted bool   `json:"deleted"`
}

type Transaction struct {
	ID           string     `json:"id"`
	Date         *time.Time `json:"date"`
	Amount       int64      `json:"amount"`
	Memo         string     `json:"memo"`
	AccountID    string     `json:"account_id"`
	AccountName  string     `json:"account_name"`
	PayeeID      *string    `json:"payee_id"`
	PayeeName    string     `json:"payee_name"`
	CategoryID   *string    `json:"category_id"`
	CategoryName string     `json:"category_name"`
	Deleted      bool       `json:"deleted"`
}

type WeeklyData struct {
	Budget       *Budget
	Categories   []Category
	Transactions []Transaction
	WeekStart    time.Time
	WeekEnd      time.Time
}

type MonthlyData struct {
	Budget       *Budget
	Categories   []Category
	Transactions []Transaction
	MonthStart   time.Time
	MonthEnd     time.Time
}

