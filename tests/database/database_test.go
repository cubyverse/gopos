package database_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopos/database"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) (*sql.DB, string, func()) {
	// Create temporary directory for test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Open test database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Initialize database schema
	if err := database.InitDB(db); err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	// Debug: Check if tables were created
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check users table: %v", err)
	}
	t.Logf("Users table count: %d", count)

	// Return cleanup function
	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, dbPath, cleanup
}

func TestUserOperations(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	// Test user creation
	t.Run("CreateUser", func(t *testing.T) {
		now := time.Now()
		result, err := db.Exec(`
			INSERT INTO users (card_number, name, role, balance, created_at)
			VALUES (?, ?, ?, ?, ?)
		`, "TEST123", "Test User", "customer", 100.0, now)
		if err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			t.Fatalf("Failed to get last insert id: %v", err)
		}

		// Verify user was created
		var user struct {
			ID         int64
			CardNumber string
			Name       string
			Role       string
			Balance    float64
		}
		err = db.QueryRow(`
			SELECT id, card_number, name, role, balance 
			FROM users 
			WHERE id = ?
		`, id).Scan(&user.ID, &user.CardNumber, &user.Name, &user.Role, &user.Balance)
		if err != nil {
			t.Fatalf("Failed to query test user: %v", err)
		}

		if user.CardNumber != "TEST123" || user.Name != "Test User" || user.Role != "customer" || user.Balance != 100.0 {
			t.Errorf("User data mismatch: got %+v", user)
		}
	})

	// Test user balance update
	t.Run("UpdateUserBalance", func(t *testing.T) {
		_, err := db.Exec(`
			UPDATE users 
			SET balance = balance + ? 
			WHERE card_number = ?
		`, 50.0, "TEST123")
		if err != nil {
			t.Fatalf("Failed to update user balance: %v", err)
		}

		var balance float64
		err = db.QueryRow("SELECT balance FROM users WHERE card_number = ?", "TEST123").Scan(&balance)
		if err != nil {
			t.Fatalf("Failed to query updated balance: %v", err)
		}

		if balance != 150.0 {
			t.Errorf("Balance mismatch: got %f, want %f", balance, 150.0)
		}
	})
}

func TestProductOperations(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	// Test product creation
	t.Run("CreateProduct", func(t *testing.T) {
		now := time.Now()
		result, err := db.Exec(`
			INSERT INTO products (barcode, name, price, created_at)
			VALUES (?, ?, ?, ?)
		`, "123456789", "Test Product", 9.99, now)
		if err != nil {
			t.Fatalf("Failed to create test product: %v", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			t.Fatalf("Failed to get last insert id: %v", err)
		}

		// Verify product was created
		var product struct {
			ID      int64
			Barcode string
			Name    string
			Price   float64
		}
		err = db.QueryRow(`
			SELECT id, barcode, name, price 
			FROM products 
			WHERE id = ?
		`, id).Scan(&product.ID, &product.Barcode, &product.Name, &product.Price)
		if err != nil {
			t.Fatalf("Failed to query test product: %v", err)
		}

		if product.Barcode != "123456789" || product.Name != "Test Product" || product.Price != 9.99 {
			t.Errorf("Product data mismatch: got %+v", product)
		}
	})

	// Test product price update
	t.Run("UpdateProductPrice", func(t *testing.T) {
		_, err := db.Exec(`
			UPDATE products 
			SET price = ? 
			WHERE barcode = ?
		`, 10.99, "123456789")
		if err != nil {
			t.Fatalf("Failed to update product price: %v", err)
		}

		var price float64
		err = db.QueryRow("SELECT price FROM products WHERE barcode = ?", "123456789").Scan(&price)
		if err != nil {
			t.Fatalf("Failed to query updated price: %v", err)
		}

		if price != 10.99 {
			t.Errorf("Price mismatch: got %f, want %f", price, 10.99)
		}
	})
}

func TestTransactionOperations(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup test data
	now := time.Now()

	// Create test user
	_, err := db.Exec(`
		INSERT INTO users (card_number, name, role, balance, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, "TEST123", "Test User", "customer", 100.0, now)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Debug: Check if user was created
	var userID int64
	err = db.QueryRow("SELECT id FROM users WHERE card_number = ?", "TEST123").Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to verify test user creation: %v", err)
	}
	t.Logf("Created test user with ID: %d", userID)

	// Create test cashier
	_, err = db.Exec(`
		INSERT INTO users (card_number, name, role, balance, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, "CASH123", "Test Cashier", "cashier", 0.0, now)
	if err != nil {
		t.Fatalf("Failed to create test cashier: %v", err)
	}

	// Debug: Check if cashier was created
	var cashierID int64
	err = db.QueryRow("SELECT id FROM users WHERE card_number = ?", "CASH123").Scan(&cashierID)
	if err != nil {
		t.Fatalf("Failed to verify test cashier creation: %v", err)
	}
	t.Logf("Created test cashier with ID: %d", cashierID)

	// Create test product
	_, err = db.Exec(`
		INSERT INTO products (barcode, name, price, created_at)
		VALUES (?, ?, ?, ?)
	`, "123456789", "Test Product", 10.0, now)
	if err != nil {
		t.Fatalf("Failed to create test product: %v", err)
	}

	// Debug: Check if product was created
	var productID int64
	err = db.QueryRow("SELECT id FROM products WHERE barcode = ?", "123456789").Scan(&productID)
	if err != nil {
		t.Fatalf("Failed to verify test product creation: %v", err)
	}
	t.Logf("Created test product with ID: %d", productID)

	// Test transaction creation
	t.Run("CreateTransaction", func(t *testing.T) {
		// Start transaction
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		// Get user's current balance
		var currentBalance float64
		err = tx.QueryRow("SELECT balance FROM users WHERE id = ?", userID).Scan(&currentBalance)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to get user balance: %v", err)
		}
		t.Logf("Current user balance: %f", currentBalance)

		// Check if user has sufficient balance
		total := 20.0 // 2 items × 10.0
		if currentBalance < total {
			tx.Rollback()
			t.Fatalf("Insufficient balance: got %f, need %f", currentBalance, total)
		}

		// Update user balance
		newBalance := currentBalance - total
		_, err = tx.Exec(`
			UPDATE users 
			SET balance = ? 
			WHERE id = ?
		`, newBalance, userID)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to update user balance: %v", err)
		}

		// Create transaction record
		result, err := tx.Exec(`
			INSERT INTO transactions (user_id, cashier_id, total, created_at)
			VALUES (?, ?, ?, ?)
		`, userID, cashierID, total, now)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to create transaction: %v", err)
		}

		transactionID, err := result.LastInsertId()
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to get transaction id: %v", err)
		}

		// Add transaction items
		_, err = tx.Exec(`
			INSERT INTO transaction_items (transaction_id, product_id, quantity, price)
			VALUES (?, ?, ?, ?)
		`, transactionID, productID, 2, 10.0)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to create transaction item: %v", err)
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			t.Fatalf("Failed to commit transaction: %v", err)
		}

		// Verify transaction was created
		var transaction struct {
			ID        int64
			UserID    int64
			CashierID int64
			Total     float64
		}
		err = db.QueryRow(`
			SELECT id, user_id, cashier_id, total 
			FROM transactions 
			WHERE id = ?
		`, transactionID).Scan(&transaction.ID, &transaction.UserID, &transaction.CashierID, &transaction.Total)
		if err != nil {
			t.Fatalf("Failed to query transaction: %v", err)
		}

		if transaction.UserID != userID || transaction.CashierID != cashierID || transaction.Total != 20.0 {
			t.Errorf("Transaction data mismatch: got %+v", transaction)
		}

		// Verify user balance was updated
		var balance float64
		err = db.QueryRow("SELECT balance FROM users WHERE id = ?", userID).Scan(&balance)
		if err != nil {
			t.Fatalf("Failed to query user balance: %v", err)
		}

		if balance != 80.0 {
			t.Errorf("Balance mismatch: got %f, want %f", balance, 80.0)
		}
	})
}
