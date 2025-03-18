package handlers

import (
	"database/sql"
	"gopos/components"
	"net/http"
)

// HandleUsers displays all users
func HandleUsers(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user from session
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

		// Get all users
		rows, err := db.Query(`
            SELECT id, name, role, card_number, balance, email, created_at 
            FROM users 
            ORDER BY created_at DESC
        `)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var users []components.User
		for rows.Next() {
			var user components.User
			var email sql.NullString
			err := rows.Scan(&user.ID, &user.Name, &user.Role, &user.CardNumber, &user.Balance, &email, &user.CreatedAt)
			if err != nil {
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			user.Email = email.String
			users = append(users, user)
		}

		data := components.UsersData{
			Title:     "Benutzerverwaltung",
			UserName:  userName,
			Role:      userRole,
			CSRFToken: generateCSRFToken(),
			Users:     users,
		}

		components.Users(data).Render(r.Context(), w)
	}
}
