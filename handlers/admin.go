package handlers

import (
	"database/sql"
	"fmt"
	"gopos/components"
	"gopos/services"
	"net/http"
	"strconv"
	"time"
)

// HandleAuditTrail displays the audit log
func HandleAuditTrail(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user info from session
		session, err := store.Get(r, sessionName)
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

		// Parse pagination parameters
		page := 1
		pageSize := 10 // Number of items per page

		// Parse page parameter if provided
		pageParam := r.URL.Query().Get("page")
		if pageParam != "" {
			parsedPage, err := strconv.Atoi(pageParam)
			if err == nil && parsedPage > 0 {
				page = parsedPage
			}
		}

		// Get total count of audit entries
		totalCount, err := getAuditEntriesCount(db)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Calculate total pages
		totalPages := (totalCount + pageSize - 1) / pageSize // Ceiling division

		// Get audit entries from database with pagination
		entries, err := getAuditEntries(db, page, pageSize)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		data := components.AuditData{
			Title:       "Audit Log",
			UserName:    userName,
			Role:        userRole,
			CSRFToken:   generateCSRFToken(),
			Entries:     entries,
			CurrentPage: page,
			TotalPages:  totalPages,
			PageSize:    pageSize,
			TotalCount:  totalCount,
		}
		components.Audit(data).Render(r.Context(), w)
	}
}

// HandleNewUser displays and processes the new user form
func HandleNewUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			data := components.UserFormData{
				Title:     "Neuer Benutzer",
				CSRFToken: generateCSRFToken(),
			}
			components.UserForm(data).Render(r.Context(), w)
			return
		}

		if r.Method == "POST" {
			// Parse form
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Error parsing form", http.StatusBadRequest)
				return
			}

			// Get form values
			cardNumber := r.FormValue("card_number")
			name := r.FormValue("name")
			role := r.FormValue("role")
			email := r.FormValue("email")

			// Validate required fields
			if cardNumber == "" || name == "" || role == "" {
				data := components.UserFormData{
					Title:     "Neuer Benutzer",
					Error:     "Bitte füllen Sie alle Pflichtfelder aus",
					CSRFToken: generateCSRFToken(),
				}
				components.UserForm(data).Render(r.Context(), w)
				return
			}

			// Start transaction
			tx, err := db.Begin()
			if err != nil {
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			defer tx.Rollback()

			// Check if card number already exists
			var existingID int
			err = tx.QueryRow("SELECT id FROM users WHERE card_number = ?", cardNumber).Scan(&existingID)
			if err != sql.ErrNoRows {
				data := components.UserFormData{
					Title:     "Neuer Benutzer",
					Error:     "Diese Kartennummer existiert bereits",
					CSRFToken: generateCSRFToken(),
				}
				components.UserForm(data).Render(r.Context(), w)
				return
			}

			// Insert new user
			if _, err = tx.Exec(`
				INSERT INTO users (card_number, name, role, email, created_at)
				VALUES (?, ?, ?, ?, ?)
			`, cardNumber, name, role, email, time.Now()); err != nil {
				data := components.UserFormData{
					Title:     "Neuer Benutzer",
					Error:     "Fehler beim Erstellen des Benutzers",
					CSRFToken: generateCSRFToken(),
				}
				components.UserForm(data).Render(r.Context(), w)
				return
			}

			// Log the action
			adminUser := r.Context().Value(contextUserKey).(components.User)
			_, err = tx.Exec(`
				INSERT INTO audit_log (user_id, action, details, created_at)
				VALUES (?, ?, ?, ?)
			`, adminUser.ID, "create_user", fmt.Sprintf("Benutzer erstellt: %s (Rolle: %s)", name, role), time.Now())

			if err != nil {
				http.Error(w, "Error logging action", http.StatusInternalServerError)
				return
			}

			// Commit transaction
			if err := tx.Commit(); err != nil {
				http.Error(w, "Error committing transaction", http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, fmt.Sprintf("/users?message=Benutzer %s erfolgreich erstellt", name), http.StatusSeeOther)
		}
	}
}

// HandleEditUser displays and processes the edit user form
func HandleEditUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			// Get user ID from query string
			userIDStr := r.URL.Query().Get("id")
			if userIDStr == "" {
				http.Error(w, "User ID is required", http.StatusBadRequest)
				return
			}

			userID, err := strconv.Atoi(userIDStr)
			if err != nil {
				http.Error(w, "Invalid user ID", http.StatusBadRequest)
				return
			}

			// Get user from database
			user, err := services.GetUserByID(db, userID)
			if err != nil {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}

			data := components.UserFormData{
				Title:     "Benutzer bearbeiten",
				User:      user,
				CSRFToken: generateCSRFToken(),
			}
			components.UserForm(data).Render(r.Context(), w)
			return
		}

		if r.Method == "POST" {
			// Parse form
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Error parsing form", http.StatusBadRequest)
				return
			}

			// Get form values
			userIDStr := r.URL.Query().Get("id")
			cardNumber := r.FormValue("card_number")
			name := r.FormValue("name")
			role := r.FormValue("role")
			email := r.FormValue("email")
			balanceStr := r.FormValue("balance")

			userID, err := strconv.Atoi(userIDStr)
			if err != nil {
				http.Error(w, "Invalid user ID", http.StatusBadRequest)
				return
			}

			// Validate required fields
			if cardNumber == "" || name == "" || role == "" {
				data := components.UserFormData{
					Title:     "Benutzer bearbeiten",
					Error:     "Bitte füllen Sie alle Pflichtfelder aus",
					CSRFToken: generateCSRFToken(),
				}
				components.UserForm(data).Render(r.Context(), w)
				return
			}

			// Parse balance
			balance, err := strconv.ParseFloat(balanceStr, 64)
			if err != nil {
				data := components.UserFormData{
					Title:     "Benutzer bearbeiten",
					Error:     "Ungültiger Betrag",
					CSRFToken: generateCSRFToken(),
				}
				components.UserForm(data).Render(r.Context(), w)
				return
			}

			// Start transaction
			tx, err := db.Begin()
			if err != nil {
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			defer tx.Rollback()

			// Check if card number already exists for different user
			var existingID int
			err = tx.QueryRow("SELECT id FROM users WHERE card_number = ? AND id != ?", cardNumber, userID).Scan(&existingID)
			if err != sql.ErrNoRows {
				data := components.UserFormData{
					Title:     "Benutzer bearbeiten",
					Error:     "Diese Kartennummer existiert bereits",
					CSRFToken: generateCSRFToken(),
				}
				components.UserForm(data).Render(r.Context(), w)
				return
			}

			// Update user
			result, err := tx.Exec(`
				UPDATE users 
				SET card_number = ?, name = ?, role = ?, email = ?, balance = ?
				WHERE id = ?
			`, cardNumber, name, role, email, balance, userID)

			if err != nil {
				data := components.UserFormData{
					Title:     "Benutzer bearbeiten",
					Error:     "Fehler beim Aktualisieren des Benutzers",
					CSRFToken: generateCSRFToken(),
				}
				components.UserForm(data).Render(r.Context(), w)
				return
			}

			rowsAffected, err := result.RowsAffected()
			if err != nil || rowsAffected != 1 {
				data := components.UserFormData{
					Title:     "Benutzer bearbeiten",
					Error:     "Benutzer nicht gefunden",
					CSRFToken: generateCSRFToken(),
				}
				components.UserForm(data).Render(r.Context(), w)
				return
			}

			// Log the action
			adminUser := r.Context().Value(contextUserKey).(components.User)
			_, err = tx.Exec(`
				INSERT INTO audit_log (user_id, action, details, created_at)
				VALUES (?, ?, ?, ?)
			`, adminUser.ID, "edit_user", fmt.Sprintf("Benutzer bearbeitet: %s (Rolle: %s)", name, role), time.Now())

			if err != nil {
				http.Error(w, "Error logging action", http.StatusInternalServerError)
				return
			}

			// Commit transaction
			if err := tx.Commit(); err != nil {
				http.Error(w, "Error committing transaction", http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, fmt.Sprintf("/users?message=Benutzer %s erfolgreich aktualisiert", name), http.StatusSeeOther)
		}
	}
}

// HandleDeleteUser processes user deletion
func HandleDeleteUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userID, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)

		// Get user name for logging
		var userName string
		err := db.QueryRow("SELECT name FROM users WHERE id = ?", userID).Scan(&userName)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Check if user has any transactions
		var transactionCount int
		err = db.QueryRow("SELECT COUNT(*) FROM transactions WHERE user_id = ?", userID).Scan(&transactionCount)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if transactionCount > 0 {
			http.Error(w, "Cannot delete user with existing transactions", http.StatusBadRequest)
			return
		}

		// Delete the user
		_, err = db.Exec("DELETE FROM users WHERE id = ?", userID)
		if err != nil {
			http.Error(w, "Error deleting user", http.StatusInternalServerError)
			return
		}

		// Log the deletion
		adminUser := r.Context().Value(contextUserKey).(components.User)
		_, err = db.Exec(`
			INSERT INTO audit_log (user_id, action, details, created_at)
			VALUES (?, ?, ?, ?)
		`, adminUser.ID, "delete_user", fmt.Sprintf("Benutzer gelöscht: %s", userName), time.Now())
		if err != nil {
			http.Error(w, "Error logging action", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/users?message=Benutzer erfolgreich gelöscht", http.StatusSeeOther)
	}
}

// HandleTopupUser handles the balance top-up form
func HandleTopupUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user from session
		session, _ := store.Get(r, "pos-session")
		userName := session.Values["name"].(string)
		userRole := session.Values["role"].(string)
		userID := session.Values["user_id"].(int)

		// Get user balance
		var balance float64
		err := db.QueryRow("SELECT balance FROM users WHERE id = ?", userID).Scan(&balance)
		if err != nil {
			http.Error(w, "Error loading user balance", http.StatusInternalServerError)
			return
		}

		if r.Method == http.MethodGet {
			// Get user ID from query parameter
			userIDStr := r.URL.Query().Get("id")
			var selectedUser *components.User
			var preselectedUserID int

			if userIDStr != "" {
				preselectedUserID, err = strconv.Atoi(userIDStr)
				if err != nil {
					http.Error(w, "Invalid user ID", http.StatusBadRequest)
					return
				}

				// Get user details
				selectedUser, err = services.GetUserByID(db, preselectedUserID)
				if err != nil {
					http.Error(w, "User not found", http.StatusNotFound)
					return
				}
			}

			data := components.TopupData{
				Title:             "Guthaben aufladen",
				UserName:          userName,
				Role:              userRole,
				Balance:           balance,
				CSRFToken:         generateCSRFToken(),
				User:              selectedUser,
				PreselectedUserID: preselectedUserID,
			}
			components.TopupForm(data).Render(r.Context(), w)
			return
		}

		if r.Method == http.MethodPost {
			var targetUserID int
			var amount float64
			var selectedUser *components.User

			// Parse form
			if err := r.ParseForm(); err != nil {
				http.Redirect(w, r, "/dashboard?error=Ungültige Anfrage", http.StatusSeeOther)
				return
			}

			// Get amount
			amountStr := r.FormValue("amount")
			amount, err = strconv.ParseFloat(amountStr, 64)
			if err != nil || amount <= 0 {
				http.Redirect(w, r, "/dashboard?error=Bitte geben Sie einen gültigen Betrag ein", http.StatusSeeOther)
				return
			}

			// Get user ID either from form or card number
			userIDStr := r.FormValue("user_id")
			if userIDStr != "" {
				targetUserID, err = strconv.Atoi(userIDStr)
				if err != nil {
					http.Redirect(w, r, "/dashboard?error=Ungültige Benutzer-ID", http.StatusSeeOther)
					return
				}
				selectedUser, err = services.GetUserByID(db, targetUserID)
				if err != nil {
					http.Redirect(w, r, "/dashboard?error=Benutzer nicht gefunden", http.StatusSeeOther)
					return
				}
			} else {
				// Look up user by card number
				cardNumber := r.FormValue("card_number")
				if cardNumber == "" {
					http.Redirect(w, r, "/dashboard?error=Bitte geben Sie eine Kartennummer ein", http.StatusSeeOther)
					return
				}

				selectedUser, err = services.GetUserByCardNumber(db, cardNumber)
				if err != nil {
					http.Redirect(w, r, "/dashboard?error=Benutzer nicht gefunden", http.StatusSeeOther)
					return
				}
				targetUserID = selectedUser.ID
			}

			// Start transaction
			tx, err := db.Begin()
			if err != nil {
				http.Redirect(w, r, "/dashboard?error=Datenbankfehler", http.StatusSeeOther)
				return
			}
			defer tx.Rollback()

			// Update balance
			result, err := tx.Exec("UPDATE users SET balance = balance + ? WHERE id = ?", amount, targetUserID)
			if err != nil {
				http.Redirect(w, r, "/dashboard?error=Fehler beim Aufladen des Guthabens", http.StatusSeeOther)
				return
			}

			rowsAffected, err := result.RowsAffected()
			if err != nil || rowsAffected != 1 {
				http.Redirect(w, r, "/dashboard?error=Fehler beim Aufladen des Guthabens", http.StatusSeeOther)
				return
			}

			// Get cashier from context
			cashierUser := r.Context().Value(contextUserKey).(components.User)

			// Log the action
			_, err = tx.Exec(`
				INSERT INTO audit_log (user_id, action, details, created_at)
				VALUES (?, ?, ?, ?)
			`, cashierUser.ID, "balance_topup", fmt.Sprintf("Guthaben aufgeladen für %s: %.2f€", selectedUser.Name, amount), time.Now())
			if err != nil {
				http.Redirect(w, r, "/dashboard?error=Fehler beim Protokollieren", http.StatusSeeOther)
				return
			}

			// Commit transaction
			if err := tx.Commit(); err != nil {
				http.Redirect(w, r, "/dashboard?error=Fehler beim Abschließen der Transaktion", http.StatusSeeOther)
				return
			}

			// Redirect with success message
			http.Redirect(w, r, fmt.Sprintf("/dashboard?success=true&message=Guthaben von %.2f€ wurde erfolgreich für %s aufgeladen", amount, selectedUser.Name), http.StatusSeeOther)
			return
		}
	}
}

// HandleUserSearch handles the search functionality for users
func HandleUserSearch(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		query = "%" + query + "%"

		rows, err := db.Query(`
			SELECT id, name, card_number, role, balance, email, created_at 
			FROM users 
			WHERE name LIKE ? OR card_number LIKE ?
			ORDER BY created_at DESC
		`, query, query)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var users []components.User
		for rows.Next() {
			var user components.User
			var email sql.NullString
			err := rows.Scan(&user.ID, &user.Name, &user.CardNumber, &user.Role, &user.Balance, &email, &user.CreatedAt)
			if err != nil {
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			user.Email = email.String
			users = append(users, user)
		}

		data := components.UsersData{
			Title:     "Benutzerverwaltung",
			Users:     users,
			CSRFToken: generateCSRFToken(),
		}
		components.UsersGrid(data).Render(r.Context(), w)
	}
}

// HandleUserFilter handles the role filtering and sorting functionality for users
func HandleUserFilter(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role := r.URL.Query().Get("role")
		sort := r.URL.Query().Get("sort")

		query := `
			SELECT id, name, card_number, role, balance, email, created_at 
			FROM users 
			WHERE 1=1
		`
		args := []interface{}{}

		if role != "" {
			query += " AND role = ?"
			args = append(args, role)
		}

		query += " ORDER BY "
		switch sort {
		case "name":
			query += "name ASC"
		case "balance":
			query += "balance DESC"
		default:
			query += "created_at DESC"
		}

		rows, err := db.Query(query, args...)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var users []components.User
		for rows.Next() {
			var user components.User
			var emailNull sql.NullString
			err := rows.Scan(&user.ID, &user.Name, &user.CardNumber, &user.Role, &user.Balance, &emailNull, &user.CreatedAt)
			if err != nil {
				println("Error scanning user in filter:", err.Error())
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			user.Email = emailNull.String
			users = append(users, user)
		}

		data := components.UsersData{
			Title:     "Benutzerverwaltung",
			Users:     users,
			CSRFToken: generateCSRFToken(),
		}
		components.UsersGrid(data).Render(r.Context(), w)
	}
}

// getAuditEntries retrieves audit entries from the database with pagination
func getAuditEntries(db *sql.DB, page, pageSize int) ([]components.AuditEntry, error) {
	offset := (page - 1) * pageSize

	rows, err := db.Query(`
		SELECT a.created_at, u.name, a.user_id, a.action, a.details
		FROM audit_log a
		LEFT JOIN users u ON a.user_id = u.id
		ORDER BY a.created_at DESC
		LIMIT ? OFFSET ?
	`, pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []components.AuditEntry
	for rows.Next() {
		var entry components.AuditEntry
		err := rows.Scan(&entry.CreatedAt, &entry.UserName, &entry.UserID, &entry.Action, &entry.Details)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// getAuditEntriesCount returns the total count of audit entries
func getAuditEntriesCount(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM audit_log").Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
