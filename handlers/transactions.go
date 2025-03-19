package handlers

import (
	"database/sql"
	"gopos/components"
	"gopos/services"
	"log"
	"net/http"
	"strconv"
	"time"
)

// HandleTransactions displays the transactions page for the logged-in user
func HandleTransactions(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user from session
		session, _ := store.Get(r, "pos-session")
		userID := session.Values["user_id"].(int)
		userName := session.Values["name"].(string)
		userRole := session.Values["role"].(string)

		// Get user balance
		var balance float64
		err := db.QueryRow("SELECT balance FROM users WHERE id = ?", userID).Scan(&balance)
		if err != nil {
			http.Error(w, "Error loading user balance", http.StatusInternalServerError)
			return
		}

		// Get transactions from database with user and cashier names
		rows, err := db.Query(`
			SELECT 
				t.id,
				u.name as user_name,
				c.name as cashier_name,
				t.total,
				datetime(t.created_at, 'localtime') as created_at
			FROM transactions t
			JOIN users u ON t.user_id = u.id
			JOIN users c ON t.cashier_id = c.id
			WHERE t.user_id = ?
			ORDER BY t.created_at DESC
			LIMIT 50
		`, userID)
		if err != nil {
			http.Error(w, "Error loading transactions", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var transactions []components.Transaction
		for rows.Next() {
			var t components.Transaction
			err := rows.Scan(&t.ID, &t.UserName, &t.CashierName, &t.Total, &t.CreatedAt)
			if err != nil {
				continue
			}

			// Load transaction items
			itemRows, err := db.Query(`
				SELECT 
					p.name,
					ti.quantity,
					ti.price
				FROM transaction_items ti
				JOIN products p ON p.id = ti.product_id
				WHERE ti.transaction_id = ?
				ORDER BY p.name
			`, t.ID)
			if err != nil {
				continue
			}
			defer itemRows.Close()

			var items []components.TransactionItem
			for itemRows.Next() {
				var item components.TransactionItem
				err := itemRows.Scan(&item.ProductName, &item.Quantity, &item.Price)
				if err != nil {
					continue
				}
				items = append(items, item)
			}
			t.Items = items

			transactions = append(transactions, t)
		}

		data := components.TransactionsData{
			Title:        "Meine Transaktionen",
			UserName:     userName,
			Role:         userRole,
			Balance:      balance,
			CSRFToken:    generateCSRFToken(),
			Transactions: transactions,
		}

		err = components.Transactions(data).Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Error rendering transactions", http.StatusInternalServerError)
			return
		}
	}
}

// HandleTransaction processes a new transaction
func HandleTransaction(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get form values
		userIDStr := r.FormValue("user_id")
		amountStr := r.FormValue("amount")
		description := r.FormValue("description")

		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			http.Error(w, "Invalid amount", http.StatusBadRequest)
			return
		}

		// Start transaction
		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// Get user's current balance and email
		var currentBalance float64
		var userName string
		var userEmail string
		err = tx.QueryRow("SELECT balance, name, email FROM users WHERE id = ?", userID).Scan(&currentBalance, &userName, &userEmail)
		if err != nil {
			log.Printf("[TRANSACTION] Error getting user data: %v", err)
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		log.Printf("[TRANSACTION] Processing transaction for user %s (ID: %d, Email: %s)", userName, userID, userEmail)

		// Calculate new balance
		newBalance := currentBalance + amount
		log.Printf("[TRANSACTION] Updating balance from %.2f€ to %.2f€", currentBalance, newBalance)

		// Update user's balance
		_, err = tx.Exec("UPDATE users SET balance = ? WHERE id = ?", newBalance, userID)
		if err != nil {
			log.Printf("[TRANSACTION] Error updating balance: %v", err)
			http.Error(w, "Error updating balance", http.StatusInternalServerError)
			return
		}

		// Record transaction
		cashier := r.Context().Value(userKey).(components.User)
		_, err = tx.Exec(`
			INSERT INTO transactions (user_id, cashier_id, total, description, created_at)
			VALUES (?, ?, ?, ?, ?)
		`, userID, cashier.ID, amount, description, time.Now())
		if err != nil {
			log.Printf("[TRANSACTION] Error recording transaction: %v", err)
			http.Error(w, "Error recording transaction", http.StatusInternalServerError)
			return
		}

		// Log the transaction
		_, err = tx.Exec(`
			INSERT INTO audit_log (user_id, action, details, created_at)
			VALUES (?, ?, ?, ?)
		`, cashier.ID, "transaction", description, time.Now())
		if err != nil {
			log.Printf("[TRANSACTION] Error logging to audit: %v", err)
			http.Error(w, "Error logging transaction", http.StatusInternalServerError)
			return
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			log.Printf("[TRANSACTION] Error committing transaction: %v", err)
			http.Error(w, "Error committing transaction", http.StatusInternalServerError)
			return
		}

		log.Printf("[TRANSACTION] Transaction completed successfully, sending email notification...")

		// Send email notification if user has email
		if userEmail != "" {
			// Use the transaction-specific email template
			if err := services.SendTransactionEmail(userEmail, userName, amount, newBalance, []services.Product{}); err != nil {
				log.Printf("[TRANSACTIONS] Error sending email notification: %v", err)
			} else {
				log.Printf("[TRANSACTIONS] Email notification sent successfully")
			}
		}

		// Return success response
		w.WriteHeader(http.StatusOK)
		log.Printf("[TRANSACTION] Request completed successfully")
	}
}
