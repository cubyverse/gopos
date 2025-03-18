package handlers

import (
	"database/sql"
	"gopos/components"
	"log"
	"net/http"
)

type ProductStats struct {
	Name     string
	Quantity int
	Revenue  float64
}

// HandleStats renders the statistics page
func HandleStats(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user info from session
		session, err := store.Get(r, sessionName)
		if err != nil {
			log.Printf("Stats error: failed to get session: %v", err)
			http.Error(w, "Session error", http.StatusInternalServerError)
			return
		}

		// Type assert session values with proper error checking
		userName, ok := session.Values["name"].(string)
		if !ok {
			log.Printf("Stats error: name not found in session or wrong type")
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		userRole, ok := session.Values["role"].(string)
		if !ok {
			log.Printf("Stats error: role not found in session or wrong type")
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// Get daily revenue
		var dailyRevenue float64
		err = db.QueryRow(`
            SELECT COALESCE(SUM(total), 0)
            FROM transactions
            WHERE DATE(created_at) = DATE('now')
        `).Scan(&dailyRevenue)
		if err != nil {
			log.Printf("Stats error: failed to get daily revenue: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Get monthly revenue
		var monthlyRevenue float64
		err = db.QueryRow(`
            SELECT COALESCE(SUM(total), 0)
            FROM transactions
            WHERE strftime('%Y-%m', created_at) = strftime('%Y-%m', 'now')
        `).Scan(&monthlyRevenue)
		if err != nil {
			log.Printf("Stats error: failed to get monthly revenue: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Get total revenue (all time)
		var totalRevenue float64
		err = db.QueryRow(`
            SELECT COALESCE(SUM(total), 0)
            FROM transactions
        `).Scan(&totalRevenue)
		if err != nil {
			log.Printf("Stats error: failed to get total revenue: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Get total system balance (sum of all user balances)
		var systemBalance float64
		err = db.QueryRow(`
            SELECT COALESCE(SUM(balance), 0)
            FROM users
        `).Scan(&systemBalance)
		if err != nil {
			log.Printf("Stats error: failed to get system balance: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Get top selling products
		topProducts, err := getProductStats(db, "DESC", 5)
		if err != nil {
			log.Printf("Stats error: failed to get top products: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Get low selling products
		lowProducts, err := getProductStats(db, "ASC", 5)
		if err != nil {
			log.Printf("Stats error: failed to get low products: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		data := components.StatsData{
			Title:          "Statistiken",
			UserName:       userName,
			Role:           userRole,
			CSRFToken:      generateCSRFToken(),
			DailyRevenue:   dailyRevenue,
			MonthlyRevenue: monthlyRevenue,
			TotalRevenue:   totalRevenue,
			SystemBalance:  systemBalance,
			TopProducts:    topProducts,
			LowProducts:    lowProducts,
		}

		if err := components.Stats(data).Render(r.Context(), w); err != nil {
			log.Printf("Stats error: failed to render template: %v", err)
			http.Error(w, "Error rendering statistics", http.StatusInternalServerError)
			return
		}
	}
}

func getProductStats(db *sql.DB, order string, limit int) ([]components.ProductStats, error) {
	query := `
        SELECT 
            p.name,
            COALESCE(SUM(ti.quantity), 0) as total_quantity,
            COALESCE(SUM(ti.quantity * ti.price), 0) as total_revenue
        FROM products p
        LEFT JOIN transaction_items ti ON p.id = ti.product_id
        GROUP BY p.id, p.name
        ORDER BY total_quantity ` + order + `
        LIMIT ?
    `

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []components.ProductStats
	for rows.Next() {
		var stat components.ProductStats
		err := rows.Scan(&stat.Name, &stat.Quantity, &stat.Revenue)
		if err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return stats, nil
}
