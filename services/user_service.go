package services

import (
	"database/sql"
	"gopos/components"
)

// GetUserByID retrieves a user by their ID
func GetUserByID(db *sql.DB, id int) (*components.User, error) {
	var user components.User
	var emailNull sql.NullString
	err := db.QueryRow("SELECT id, name, card_number, role, balance, email, created_at FROM users WHERE id = ?", id).
		Scan(&user.ID, &user.Name, &user.CardNumber, &user.Role, &user.Balance, &emailNull, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	user.Email = emailNull.String
	return &user, nil
}

// GetUserByCardNumber retrieves a user by their card number
func GetUserByCardNumber(db *sql.DB, cardNumber string) (*components.User, error) {
	var user components.User
	var emailNull sql.NullString
	err := db.QueryRow("SELECT id, name, card_number, role, balance, email, created_at FROM users WHERE card_number = ?", cardNumber).
		Scan(&user.ID, &user.Name, &user.CardNumber, &user.Role, &user.Balance, &emailNull, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	user.Email = emailNull.String
	return &user, nil
}
