package handlers

import (
	"database/sql"
	"encoding/json"
	"gopos/components"
	"gopos/services"
	"log"
	"net/http"
	"time"
)

type CartItem struct {
	ProductID int64   `json:"product_id"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Quantity  int     `json:"quantity"`
}

type CheckoutRequest struct {
	CardNumber string     `json:"card_number"`
	Total      float64    `json:"total"`
	Items      []CartItem `json:"items"`
}

// HandleCheckout renders the checkout page
func HandleCheckout(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user info from session
		session, err := store.Get(r, sessionName)
		if err != nil {
			log.Printf("Checkout error: failed to get session: %v", err)
			http.Error(w, "Session error", http.StatusInternalServerError)
			return
		}

		userName, ok := session.Values["name"].(string)
		if !ok {
			log.Printf("Checkout error: name not found in session or wrong type")
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		userRole, ok := session.Values["role"].(string)
		if !ok {
			log.Printf("Checkout error: role not found in session or wrong type")
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		data := components.CheckoutData{
			Title:     "Kasse",
			UserName:  userName,
			Role:      userRole,
			CSRFToken: generateCSRFToken(),
		}

		if err := components.Checkout(data).Render(r.Context(), w); err != nil {
			log.Printf("Checkout error: failed to render template: %v", err)
			http.Error(w, "Error rendering checkout", http.StatusInternalServerError)
			return
		}
	}
}

// HandleCustomerLookup looks up a customer by card number
func HandleCustomerLookup(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cardNumber := r.URL.Query().Get("card_number")
		log.Printf("[DEBUG] Looking up customer with card number: %s", cardNumber)

		if cardNumber == "" {
			http.Error(w, "Kartennummer erforderlich", http.StatusBadRequest)
			return
		}

		user, err := services.GetUserByCardNumber(db, cardNumber)
		if err == sql.ErrNoRows {
			log.Printf("[DEBUG] No user found with card number: %s", cardNumber)
			http.Error(w, "Keine Karte mit dieser Nummer gefunden", http.StatusNotFound)
			return
		} else if err != nil {
			log.Printf("[DEBUG] Database error looking up card number %s: %v", cardNumber, err)
			http.Error(w, "Datenbankfehler beim Suchen der Karte", http.StatusInternalServerError)
			return
		}

		log.Printf("[DEBUG] Found user: %+v", user)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}
}

// HandleProductScan looks up a product by barcode
func HandleProductScan(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		barcode := r.URL.Query().Get("barcode")
		if barcode == "" {
			http.Error(w, "Barcode erforderlich", http.StatusBadRequest)
			return
		}

		var product components.Product
		err := db.QueryRow(`
			SELECT id, barcode, name, price 
			FROM products 
			WHERE barcode = ?
		`, barcode).Scan(&product.ID, &product.Barcode, &product.Name, &product.Price)

		if err == sql.ErrNoRows {
			http.Error(w, "Produkt nicht gefunden", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, "Datenbankfehler", http.StatusInternalServerError)
			return
		}

		// Format the response as a CartItem
		cartItem := CartItem{
			ProductID: int64(product.ID),
			Name:      product.Name,
			Price:     product.Price,
			Quantity:  1, // Default quantity is 1
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cartItem)
	}
}

// HandleCompleteCheckout processes the checkout
func HandleCompleteCheckout(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[CHECKOUT] Starting checkout process...")

		// Get cashier info from session
		session, err := store.Get(r, sessionName)
		if err != nil {
			log.Printf("[CHECKOUT] Session error: %v", err)
			http.Error(w, "Session error", http.StatusInternalServerError)
			return
		}

		cashierID, ok := session.Values["user_id"].(int)
		if !ok {
			log.Printf("[CHECKOUT] Cashier ID not found in session")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		cashierName, ok := session.Values["name"].(string)
		if !ok {
			log.Printf("[CHECKOUT] Cashier name not found in session")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var request CheckoutRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			log.Printf("[CHECKOUT] Error decoding request: %v", err)
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		log.Printf("[CHECKOUT] Processing checkout for user card: %s, Total: %.2f €", request.CardNumber, request.Total)

		tx, err := db.Begin()
		if err != nil {
			log.Printf("[CHECKOUT] Error starting transaction: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// Get user and check balance
		var user struct {
			ID      int64
			Name    string
			Balance float64
			Email   sql.NullString
		}
		err = tx.QueryRow(`
			SELECT id, name, balance, email 
			FROM users 
			WHERE card_number = ?`, request.CardNumber).Scan(&user.ID, &user.Name, &user.Balance, &user.Email)

		if err == sql.ErrNoRows {
			log.Printf("[CHECKOUT] User not found for card: %s", request.CardNumber)
			http.Error(w, "User not found", http.StatusNotFound)
			return
		} else if err != nil {
			log.Printf("[CHECKOUT] Error fetching user: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		log.Printf("[CHECKOUT] User found: %s (ID: %d), Current Balance: %.2f €", user.Name, user.ID, user.Balance)

		if user.Balance < request.Total {
			log.Printf("[CHECKOUT] Insufficient balance: Balance=%.2f €, Required=%.2f €", user.Balance, request.Total)
			http.Error(w, "Insufficient balance", http.StatusBadRequest)
			return
		}

		// Update balance
		newBalance := user.Balance - request.Total
		log.Printf("[CHECKOUT] Updating balance: %.2f € -> %.2f €", user.Balance, newBalance)

		result, err := tx.Exec(`
			UPDATE users 
			SET balance = ? 
			WHERE id = ?`, newBalance, user.ID)

		if err != nil {
			log.Printf("[CHECKOUT] Error updating balance: %v", err)
			http.Error(w, "Error updating balance", http.StatusInternalServerError)
			return
		}

		if rows, _ := result.RowsAffected(); rows != 1 {
			log.Printf("[CHECKOUT] Expected 1 row affected, got %d", rows)
			http.Error(w, "Error updating balance", http.StatusInternalServerError)
			return
		}

		// Record transaction
		log.Printf("[CHECKOUT] Recording transaction by cashier: %s (ID: %d)", cashierName, cashierID)

		result, err = tx.Exec(`
			INSERT INTO transactions (user_id, cashier_id, total, created_at)
			VALUES (?, ?, ?, ?)
		`, user.ID, cashierID, request.Total, time.Now())

		if err != nil {
			log.Printf("[CHECKOUT] Error recording transaction: %v", err)
			http.Error(w, "Error recording transaction", http.StatusInternalServerError)
			return
		}

		transactionID, _ := result.LastInsertId()
		log.Printf("[CHECKOUT] Transaction recorded with ID: %d", transactionID)

		// Record transaction items
		for _, item := range request.Items {
			_, err = tx.Exec(`
				INSERT INTO transaction_items (transaction_id, product_id, quantity, price)
				VALUES (?, ?, ?, ?)
			`, transactionID, item.ProductID, item.Quantity, item.Price)

			if err != nil {
				log.Printf("[CHECKOUT] Error recording transaction item: %v", err)
				http.Error(w, "Error recording transaction items", http.StatusInternalServerError)
				return
			}
		}

		// Commit transaction
		log.Printf("[CHECKOUT] Attempting to commit transaction...")
		if err := tx.Commit(); err != nil {
			log.Printf("[CHECKOUT] Error committing transaction: %v", err)
			http.Error(w, "Error completing transaction", http.StatusInternalServerError)
			return
		}

		log.Printf("[CHECKOUT] ====== TRANSACTION SUMMARY ======")
		log.Printf("[CHECKOUT] Customer: %s (ID: %d)", user.Name, user.ID)
		log.Printf("[CHECKOUT] Total Amount: %.2f €", request.Total)
		log.Printf("[CHECKOUT] New Balance: %.2f €", newBalance)
		log.Printf("[CHECKOUT] Items Count: %d", len(request.Items))
		log.Printf("[CHECKOUT] Cashier: %s (ID: %d)", cashierName, cashierID)
		log.Printf("[CHECKOUT] Transaction ID: %d", transactionID)
		log.Printf("[CHECKOUT] ================================")

		// Convert cart items to email products
		var emailProducts []services.Product
		for _, item := range request.Items {
			emailProducts = append(emailProducts, services.Product{
				Name:     item.Name,
				Price:    item.Price,
				Quantity: item.Quantity,
			})
		}

		// Send email notification if user has email
		if user.Email.Valid {
			if err := services.SendTransactionEmail(user.Email.String, user.Name, -request.Total, newBalance, emailProducts); err != nil {
				log.Printf("[CHECKOUT] Error sending email notification: %v", err)
			}
		}

		// Return success response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"balance": newBalance,
		})
	}
}
