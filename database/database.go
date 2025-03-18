package database

import (
	"database/sql"
	"log"
	"time"
)

func InitDB(db *sql.DB) error {
	// Create tables
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		card_number TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		role TEXT NOT NULL CHECK(role IN ('admin', 'cashier', 'customer')),
		balance REAL NOT NULL DEFAULT 0,
		email TEXT,
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		barcode TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		price REAL NOT NULL,
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS transactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		cashier_id INTEGER NOT NULL,
		total REAL NOT NULL,
		created_at DATETIME NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id),
		FOREIGN KEY (cashier_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS transaction_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		transaction_id INTEGER NOT NULL,
		product_id INTEGER NOT NULL,
		quantity INTEGER NOT NULL,
		price REAL NOT NULL,
		FOREIGN KEY (transaction_id) REFERENCES transactions(id),
		FOREIGN KEY (product_id) REFERENCES products(id)
	);

	CREATE TABLE IF NOT EXISTS audit_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		action TEXT NOT NULL,
		details TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	CREATE INDEX IF NOT EXISTS idx_users_card_number ON users(card_number);
	CREATE INDEX IF NOT EXISTS idx_products_barcode ON products(barcode);
	CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
	CREATE INDEX IF NOT EXISTS idx_transaction_items_transaction_id ON transaction_items(transaction_id);
	CREATE INDEX IF NOT EXISTS idx_audit_log_user_id ON audit_log(user_id);
	CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON audit_log(created_at);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return err
	}

	// Check if admin user exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&count)
	if err != nil {
		return err
	}

	// Create default admin user if none exists
	if count == 0 {
		now := time.Now()
		_, err = db.Exec(`
			INSERT INTO users (card_number, name, role, balance, created_at)
			VALUES (?, ?, ?, ?, ?)
		`, "ADMIN", "Administrator", "admin", 0.0, now)
		if err != nil {
			return err
		}
		log.Println("Created default admin user")
	}

	return nil
}
