package handlers

import (
	"database/sql"
	"fmt"
	"gopos/components"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// HandleProducts displays all products
func HandleProducts(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user from session
		session, err := store.Get(r, "pos-session")
		if err != nil {
			http.Error(w, "Session error", http.StatusInternalServerError)
			return
		}

		userName, ok := session.Values["name"].(string)
		if !ok {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		userRole, ok := session.Values["role"].(string)
		if !ok {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		userID, ok := session.Values["user_id"].(int)
		if !ok {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// Get user balance
		var balance float64
		err = db.QueryRow("SELECT balance FROM users WHERE id = ?", userID).Scan(&balance)
		if err != nil {
			http.Error(w, "Error loading user balance", http.StatusInternalServerError)
			return
		}

		rows, err := db.Query(`
            SELECT id, name, barcode, price, created_at 
            FROM products 
            ORDER BY created_at DESC
        `)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var products []components.Product
		for rows.Next() {
			var product components.Product
			err := rows.Scan(&product.ID, &product.Name, &product.Barcode, &product.Price, &product.CreatedAt)
			if err != nil {
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			products = append(products, product)
		}

		data := components.ProductsData{
			Title:     "Produktverwaltung",
			UserName:  userName,
			Role:      userRole,
			Balance:   balance,
			CSRFToken: generateCSRFToken(),
			Products:  products,
		}
		components.Products(data).Render(r.Context(), w)
	}
}

// HandleNewProduct handles the creation of new products
func HandleNewProduct(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			data := components.ProductFormData{
				Title:     "Neues Produkt",
				CSRFToken: generateCSRFToken(),
				Error:     "",
				Success:   false,
			}
			components.ProductForm(data).Render(r.Context(), w)
			return
		}

		// Handle POST request
		if err := r.ParseForm(); err != nil {
			data := components.ProductFormData{
				Title:     "Neues Produkt",
				Error:     "Fehler beim Verarbeiten des Formulars",
				CSRFToken: generateCSRFToken(),
				Success:   false,
			}
			components.ProductForm(data).Render(r.Context(), w)
			return
		}

		// Validate CSRF token
		token := r.FormValue("csrf_token")
		if !verifyCSRFToken(token) {
			data := components.ProductFormData{
				Title:     "Neues Produkt",
				Error:     "Ungültiger CSRF-Token",
				CSRFToken: generateCSRFToken(),
				Success:   false,
			}
			components.ProductForm(data).Render(r.Context(), w)
			return
		}

		// Get form values
		barcode := strings.TrimSpace(r.FormValue("barcode"))
		name := strings.TrimSpace(r.FormValue("name"))
		price, err := strconv.ParseFloat(r.FormValue("price"), 64)

		// Validate required fields
		if barcode == "" || name == "" {
			data := components.ProductFormData{
				Title:     "Neues Produkt",
				Error:     "Bitte füllen Sie alle Pflichtfelder aus",
				CSRFToken: generateCSRFToken(),
				Product:   &components.Product{Barcode: barcode, Name: name, Price: price},
				Success:   false,
			}
			components.ProductForm(data).Render(r.Context(), w)
			return
		}

		// Validate price
		if err != nil || price < 0 {
			data := components.ProductFormData{
				Title:     "Neues Produkt",
				Error:     "Bitte geben Sie einen gültigen Preis ein",
				CSRFToken: generateCSRFToken(),
				Product:   &components.Product{Barcode: barcode, Name: name, Price: price},
				Success:   false,
			}
			components.ProductForm(data).Render(r.Context(), w)
			return
		}

		// Check if barcode already exists
		var existingID int
		err = db.QueryRow("SELECT id FROM products WHERE barcode = ?", barcode).Scan(&existingID)
		if err != sql.ErrNoRows {
			data := components.ProductFormData{
				Title:     "Neues Produkt",
				Error:     "Dieser Barcode existiert bereits",
				CSRFToken: generateCSRFToken(),
				Product:   &components.Product{Barcode: barcode, Name: name, Price: price},
				Success:   false,
			}
			components.ProductForm(data).Render(r.Context(), w)
			return
		}

		// Begin transaction
		tx, err := db.Begin()
		if err != nil {
			data := components.ProductFormData{
				Title:     "Neues Produkt",
				Error:     "Datenbankfehler",
				CSRFToken: generateCSRFToken(),
				Product:   &components.Product{Barcode: barcode, Name: name, Price: price},
				Success:   false,
			}
			components.ProductForm(data).Render(r.Context(), w)
			return
		}
		defer tx.Rollback()

		// Insert new product
		result, err := tx.Exec(`
            INSERT INTO products (barcode, name, price, created_at)
            VALUES (?, ?, ?, ?)
        `, barcode, name, price, time.Now())

		if err != nil {
			data := components.ProductFormData{
				Title:     "Neues Produkt",
				Error:     "Fehler beim Speichern des Produkts",
				CSRFToken: generateCSRFToken(),
				Product:   &components.Product{Barcode: barcode, Name: name, Price: price},
				Success:   false,
			}
			components.ProductForm(data).Render(r.Context(), w)
			return
		}

		// Get the new product's ID for audit log
		productID, err := result.LastInsertId()
		if err != nil {
			data := components.ProductFormData{
				Title:     "Neues Produkt",
				Error:     "Fehler beim Speichern des Produkts",
				CSRFToken: generateCSRFToken(),
				Product:   &components.Product{Barcode: barcode, Name: name, Price: price},
				Success:   false,
			}
			components.ProductForm(data).Render(r.Context(), w)
			return
		}

		// Log the action
		adminUser := r.Context().Value(userKey).(components.User)
		_, err = tx.Exec(`
            INSERT INTO audit_log (user_id, action, details, created_at)
            VALUES (?, ?, ?, ?)
        `, adminUser.ID, "create_product", fmt.Sprintf("Produkt (ID: %d) erstellt: %s (Barcode: %s)", productID, name, barcode), time.Now())

		if err != nil {
			data := components.ProductFormData{
				Title:     "Neues Produkt",
				Error:     "Fehler beim Speichern des Audit-Logs",
				CSRFToken: generateCSRFToken(),
				Product:   &components.Product{Barcode: barcode, Name: name, Price: price},
				Success:   false,
			}
			components.ProductForm(data).Render(r.Context(), w)
			return
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			data := components.ProductFormData{
				Title:     "Neues Produkt",
				Error:     "Fehler beim Speichern des Produkts",
				CSRFToken: generateCSRFToken(),
				Product:   &components.Product{Barcode: barcode, Name: name, Price: price},
				Success:   false,
			}
			components.ProductForm(data).Render(r.Context(), w)
			return
		}

		http.Redirect(w, r, "/products", http.StatusSeeOther)
		return
	}
}

// HandleEditProduct handles the editing of existing products
func HandleEditProduct(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			// Get product ID from query parameters
			productIDStr := r.URL.Query().Get("id")
			if productIDStr == "" {
				http.Redirect(w, r, "/products", http.StatusSeeOther)
				return
			}

			productID, err := strconv.ParseInt(productIDStr, 10, 64)
			if err != nil {
				http.Error(w, "Invalid product ID", http.StatusBadRequest)
				return
			}

			// Get product from database
			var product components.Product
			err = db.QueryRow(`
                SELECT id, name, barcode, price, created_at
                FROM products
                WHERE id = ?
            `, productID).Scan(&product.ID, &product.Name, &product.Barcode, &product.Price, &product.CreatedAt)
			if err != nil {
				http.Error(w, "Product not found", http.StatusNotFound)
				return
			}

			data := components.ProductFormData{
				Title:     "Produkt bearbeiten",
				Product:   &product,
				CSRFToken: generateCSRFToken(),
				Error:     "",
				Success:   false,
			}
			components.ProductForm(data).Render(r.Context(), w)
			return
		}

		if r.Method == http.MethodPost {
			// Get product ID from query parameters
			productIDStr := r.URL.Query().Get("id")
			if productIDStr == "" {
				http.Error(w, "Product ID is required", http.StatusBadRequest)
				return
			}

			productID, err := strconv.ParseInt(productIDStr, 10, 64)
			if err != nil {
				http.Error(w, "Invalid product ID", http.StatusBadRequest)
				return
			}

			// Validate CSRF token
			token := r.FormValue("csrf_token")
			if !verifyCSRFToken(token) {
				data := components.ProductFormData{
					Title:     "Produkt bearbeiten",
					Error:     "Ungültiger CSRF-Token",
					CSRFToken: generateCSRFToken(),
					Success:   false,
				}
				components.ProductForm(data).Render(r.Context(), w)
				return
			}

			// Get form values
			barcode := strings.TrimSpace(r.FormValue("barcode"))
			name := strings.TrimSpace(r.FormValue("name"))
			price, err := strconv.ParseFloat(r.FormValue("price"), 64)

			// Create a product object to preserve form data
			product := &components.Product{
				ID:      int(productID),
				Barcode: barcode,
				Name:    name,
				Price:   price,
			}

			// Validate required fields
			if barcode == "" || name == "" {
				data := components.ProductFormData{
					Title:     "Produkt bearbeiten",
					Error:     "Bitte füllen Sie alle Pflichtfelder aus",
					CSRFToken: generateCSRFToken(),
					Product:   product,
					Success:   false,
				}
				components.ProductForm(data).Render(r.Context(), w)
				return
			}

			// Validate price
			if err != nil || price < 0 {
				data := components.ProductFormData{
					Title:     "Produkt bearbeiten",
					Error:     "Bitte geben Sie einen gültigen Preis ein",
					CSRFToken: generateCSRFToken(),
					Product:   product,
					Success:   false,
				}
				components.ProductForm(data).Render(r.Context(), w)
				return
			}

			// Check if barcode already exists for other products
			var existingID int
			err = db.QueryRow("SELECT id FROM products WHERE barcode = ? AND id != ?", barcode, productID).Scan(&existingID)
			if err != sql.ErrNoRows {
				data := components.ProductFormData{
					Title:     "Produkt bearbeiten",
					Error:     "Dieser Barcode wird bereits von einem anderen Produkt verwendet",
					CSRFToken: generateCSRFToken(),
					Product:   product,
					Success:   false,
				}
				components.ProductForm(data).Render(r.Context(), w)
				return
			}

			// Begin transaction
			tx, err := db.Begin()
			if err != nil {
				data := components.ProductFormData{
					Title:     "Produkt bearbeiten",
					Error:     "Datenbankfehler",
					CSRFToken: generateCSRFToken(),
					Product:   product,
					Success:   false,
				}
				components.ProductForm(data).Render(r.Context(), w)
				return
			}
			defer tx.Rollback()

			// Update product
			_, err = tx.Exec(`
                UPDATE products 
                SET barcode = ?, name = ?, price = ?
                WHERE id = ?
            `, barcode, name, price, productID)

			if err != nil {
				data := components.ProductFormData{
					Title:     "Produkt bearbeiten",
					Error:     "Fehler beim Aktualisieren des Produkts",
					CSRFToken: generateCSRFToken(),
					Product:   product,
					Success:   false,
				}
				components.ProductForm(data).Render(r.Context(), w)
				return
			}

			// Log the action
			adminUser := r.Context().Value(userKey).(components.User)
			_, err = tx.Exec(`
                INSERT INTO audit_log (user_id, action, details, created_at)
                VALUES (?, ?, ?, ?)
            `, adminUser.ID, "edit_product", fmt.Sprintf("Produkt (ID: %d) bearbeitet: %s (Barcode: %s)", productID, name, barcode), time.Now())

			if err != nil {
				data := components.ProductFormData{
					Title:     "Produkt bearbeiten",
					Error:     "Fehler beim Speichern des Audit-Logs",
					CSRFToken: generateCSRFToken(),
					Product:   product,
					Success:   false,
				}
				components.ProductForm(data).Render(r.Context(), w)
				return
			}

			// Commit transaction
			if err := tx.Commit(); err != nil {
				data := components.ProductFormData{
					Title:     "Produkt bearbeiten",
					Error:     "Fehler beim Aktualisieren des Produkts",
					CSRFToken: generateCSRFToken(),
					Product:   product,
					Success:   false,
				}
				components.ProductForm(data).Render(r.Context(), w)
				return
			}

			http.Redirect(w, r, "/products", http.StatusSeeOther)
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleDeleteProduct processes product deletion
func HandleDeleteProduct(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		productID, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)

		// Get product name for logging
		var productName string
		err := db.QueryRow("SELECT name FROM products WHERE id = ?", productID).Scan(&productName)
		if err != nil {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}

		// Check if product has any transactions
		var transactionCount int
		err = db.QueryRow("SELECT COUNT(*) FROM transaction_items WHERE product_id = ?", productID).Scan(&transactionCount)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if transactionCount > 0 {
			http.Error(w, "Cannot delete product with existing transactions", http.StatusBadRequest)
			return
		}

		// Delete the product
		_, err = db.Exec("DELETE FROM products WHERE id = ?", productID)
		if err != nil {
			http.Error(w, "Error deleting product", http.StatusInternalServerError)
			return
		}

		// Log the action
		adminUser := r.Context().Value(userKey).(components.User)
		_, err = db.Exec(`
            INSERT INTO audit_log (user_id, action, details, created_at)
            VALUES (?, ?, ?, ?)
        `, adminUser.ID, "delete_product", "Produkt gelöscht: "+productName, time.Now())
		if err != nil {
			http.Error(w, "Error logging action", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/products", http.StatusSeeOther)
	}
}
