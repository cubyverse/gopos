package models

import "time"

type User struct {
	ID         int64     `json:"id"`
	CardNumber string    `json:"card_number"`
	Name       string    `json:"name"`
	Role       string    `json:"role"` // admin, cashier, customer
	Balance    float64   `json:"balance"`
	Email      string    `json:"email"`
	CreatedAt  time.Time `json:"created_at"`
}

type Product struct {
	ID        int64     `json:"id"`
	Barcode   string    `json:"barcode"`
	Name      string    `json:"name"`
	Price     float64   `json:"price"`
	CreatedAt time.Time `json:"created_at"`
}

type Transaction struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	CashierID int64     `json:"cashier_id"`
	Total     float64   `json:"total"`
	CreatedAt time.Time `json:"created_at"`
}

type TransactionItem struct {
	ID            int64   `json:"id"`
	TransactionID int64   `json:"transaction_id"`
	ProductID     int64   `json:"product_id"`
	Quantity      int     `json:"quantity"`
	Price         float64 `json:"price"`
}

type AuditLog struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Action    string    `json:"action"`
	Details   string    `json:"details"`
	CreatedAt time.Time `json:"created_at"`
}
