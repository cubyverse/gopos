package handlers

import (
	"database/sql"
	"gopos/components"
	"log"
	"net/http"
)

// HandleDashboard renders the dashboard page
func HandleDashboard(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user info from session
		session, err := store.Get(r, sessionName)
		if err != nil {
			log.Printf("Dashboard error: failed to get session: %v", err)
			http.Error(w, "Session error", http.StatusInternalServerError)
			return
		}

		// Type assert session values with proper error checking
		userID, ok := session.Values["user_id"].(int)
		if !ok {
			log.Printf("Dashboard error: user_id not found in session or wrong type")
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		userName, ok := session.Values["name"].(string)
		if !ok {
			log.Printf("Dashboard error: name not found in session or wrong type")
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		userRole, ok := session.Values["role"].(string)
		if !ok {
			log.Printf("Dashboard error: role not found in session or wrong type")
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// Get user balance from database
		var balance float64
		err = db.QueryRow("SELECT balance FROM users WHERE id = ?", userID).Scan(&balance)
		if err != nil {
			log.Printf("Dashboard error: failed to get user balance: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		data := components.DashboardData{
			Title:     "Dashboard",
			Name:      userName,
			Role:      userRole,
			Balance:   balance,
			CSRFToken: generateCSRFToken(),
		}

		if err := components.Dashboard(data).Render(r.Context(), w); err != nil {
			log.Printf("Dashboard error: failed to render template: %v", err)
			http.Error(w, "Error rendering dashboard", http.StatusInternalServerError)
			return
		}
	}
}
