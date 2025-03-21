package handlers

import (
	"database/sql"
	"fmt"
	"gopos/components"
	"gopos/services"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// HandleBalanceTopup handles the balance top-up functionality
func HandleBalanceTopup(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			data := components.BalanceTopupData{
				Title:   "Guthaben aufladen",
				Success: r.URL.Query().Get("success") == "true",
				Error:   r.URL.Query().Get("error"),
			}

			// Parse amount and balance if present
			if amountStr := r.URL.Query().Get("amount"); amountStr != "" {
				if amount, err := strconv.ParseFloat(amountStr, 64); err == nil {
					data.Amount = amount
				}
			}
			if balanceStr := r.URL.Query().Get("balance"); balanceStr != "" {
				if balance, err := strconv.ParseFloat(balanceStr, 64); err == nil {
					data.Balance = balance
				}
			}

			components.BalanceTopup(data).Render(r.Context(), w)
			return
		}

		if r.Method == "POST" {
			log.Printf("[TRANSACTION] Starting balance top-up process")

			// Parse form values
			cardNumber := r.FormValue("card_number")
			amountStr := r.FormValue("amount")

			// Konvertiere Komma zu Punkt für deutsches Zahlenformat
			amountStr = strings.Replace(amountStr, ",", ".", -1)

			if cardNumber == "" {
				http.Redirect(w, r, "/balance/topup?error=Kartennummer ist erforderlich", http.StatusSeeOther)
				return
			}

			// Convert amount to float
			amount, err := strconv.ParseFloat(amountStr, 64)
			if err != nil {
				http.Redirect(w, r, "/balance/topup?error=Ungültiger Betrag", http.StatusSeeOther)
				return
			}

			if amount <= 0 {
				http.Redirect(w, r, "/balance/topup?error=Betrag muss größer als 0 sein", http.StatusSeeOther)
				return
			}

			// Start transaction
			tx, err := db.Begin()
			if err != nil {
				log.Printf("[TRANSACTION] Error starting database transaction: %v", err)
				http.Redirect(w, r, "/balance/topup?error=Datenbankfehler beim Starten der Transaktion", http.StatusSeeOther)
				return
			}
			defer tx.Rollback()

			// Get user and current balance
			var userID int64
			var currentBalance float64
			var userName string
			err = tx.QueryRow(`
				SELECT id, balance, name 
				FROM users 
				WHERE card_number = ?`, cardNumber).Scan(&userID, &currentBalance, &userName)

			if err == sql.ErrNoRows {
				log.Printf("[TRANSACTION] User not found for card number: %s", cardNumber)
				http.Redirect(w, r, "/balance/topup?error=Benutzer nicht gefunden", http.StatusSeeOther)
				return
			} else if err != nil {
				log.Printf("[TRANSACTION] Error loading user data: %v", err)
				http.Redirect(w, r, "/balance/topup?error=Fehler beim Laden des Benutzers", http.StatusSeeOther)
				return
			}

			log.Printf("[TRANSACTION] User found: ID=%d, Name=%s, Current Balance=%.2f", userID, userName, currentBalance)

			// Calculate new balance
			newBalance := currentBalance + amount

			// Update user balance
			result, err := tx.Exec(`
				UPDATE users 
				SET balance = ? 
				WHERE id = ?`, newBalance, userID)

			if err != nil {
				log.Printf("[TRANSACTION] Error updating balance: %v", err)
				http.Redirect(w, r, "/balance/topup?error=Fehler beim Aktualisieren des Guthabens", http.StatusSeeOther)
				return
			}

			rowsAffected, err := result.RowsAffected()
			if err != nil || rowsAffected != 1 {
				log.Printf("[TRANSACTION] Error with affected rows: err=%v, rows=%d", err, rowsAffected)
				http.Redirect(w, r, "/balance/topup?error=Fehler beim Aktualisieren des Guthabens", http.StatusSeeOther)
				return
			}

			log.Printf("[TRANSACTION] Balance updated successfully: New Balance=%.2f", newBalance)

			// Record transaction
			cashierUser := r.Context().Value(userKey).(components.User)
			log.Printf("[TRANSACTION] Recording transaction: Amount=%.2f, Cashier=%s (ID=%d)", amount, cashierUser.Name, cashierUser.ID)

			result, err = tx.Exec(`
				INSERT INTO transactions (user_id, cashier_id, total, description, created_at)
				VALUES (?, ?, ?, ?, ?)
			`, userID, cashierUser.ID, amount, "Guthaben aufgeladen", time.Now())

			if err != nil {
				log.Printf("[TRANSACTION] Error recording transaction: %v", err)
				http.Redirect(w, r, "/balance/topup?error=Fehler beim Speichern der Transaktion", http.StatusSeeOther)
				return
			}

			transactionID, err := result.LastInsertId()
			if err == nil {
				log.Printf("[TRANSACTION] Transaction recorded successfully: ID=%d", transactionID)
			}

			// Log the action
			_, err = tx.Exec(`
				INSERT INTO audit_log (user_id, action, details, created_at)
				VALUES (?, ?, ?, ?)
			`, cashierUser.ID, "balance_topup", fmt.Sprintf("Guthaben aufgeladen für %s: %.2f €", userName, amount), time.Now())
			if err != nil {
				http.Redirect(w, r, "/balance/topup?error=Fehler beim Speichern des Audit-Logs", http.StatusSeeOther)
				return
			}

			// Commit transaction
			log.Printf("[TRANSACTION] Attempting to commit transaction...")
			if err := tx.Commit(); err != nil {
				log.Printf("[TRANSACTION] ERROR: Failed to commit transaction: %v", err)
				http.Redirect(w, r, "/balance/topup?error=Fehler beim Abschließen der Transaktion", http.StatusSeeOther)
				return
			}
			log.Printf("[TRANSACTION] SUCCESS: Transaction committed successfully!")
			log.Printf("[TRANSACTION] ====== SUMMARY ======")
			log.Printf("[TRANSACTION] User: %s (ID: %d)", userName, userID)
			log.Printf("[TRANSACTION] Amount: %.2f €", amount)
			log.Printf("[TRANSACTION] New Balance: %.2f €", newBalance)
			log.Printf("[TRANSACTION] Cashier: %s", cashierUser.Name)
			log.Printf("[TRANSACTION] ====================")

			// Get user email
			var userEmail string
			err = db.QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&userEmail)
			if err == nil && userEmail != "" {
				// Send email notification
				go func() {
					if err := services.SendTopupEmail(userEmail, userName, amount, newBalance); err != nil {
						log.Printf("[BALANCE] Error sending top-up email: %v", err)
					} else {
						log.Printf("[BALANCE] Email notification sent successfully")
					}
				}()
			}

			// Show success message with amount and new balance
			data := components.BalanceTopupData{
				Title:   "Guthaben aufladen",
				Success: true,
				Amount:  amount,
				Balance: newBalance,
			}
			components.BalanceTopup(data).Render(r.Context(), w)
		}
	}
}
