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

		// Get daily revenue with timestamp format awareness
		var dailyRevenue float64
		err = db.QueryRow(`
            SELECT COALESCE(SUM(total), 0)
            FROM transactions
            WHERE substr(created_at, 1, 10) = substr(datetime('now', 'localtime'), 1, 10)
        `).Scan(&dailyRevenue)
		if err != nil {
			log.Printf("Stats error: failed to get daily revenue: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Get monthly revenue with timestamp format awareness
		var monthlyRevenue float64
		err = db.QueryRow(`
            SELECT COALESCE(SUM(total), 0)
            FROM transactions
            WHERE substr(created_at, 1, 7) = substr(datetime('now', 'localtime'), 1, 7)
        `).Scan(&monthlyRevenue)
		if err != nil {
			log.Printf("Stats error: failed to get monthly revenue: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Get transaction dates to debug
		rows, err := db.Query(`
			SELECT 
				created_at, 
				total,
				substr(created_at, 1, 10) as day_part,
				substr(datetime('now', 'localtime'), 1, 10) as today_part,
				substr(created_at, 1, 7) as month_part,
				substr(datetime('now', 'localtime'), 1, 7) as current_month_part,
				CASE WHEN substr(created_at, 1, 10) = substr(datetime('now', 'localtime'), 1, 10) THEN 'YES' ELSE 'NO' END as is_today,
				CASE WHEN substr(created_at, 1, 7) = substr(datetime('now', 'localtime'), 1, 7) THEN 'YES' ELSE 'NO' END as is_this_month
			FROM transactions 
			ORDER BY created_at DESC 
			LIMIT 3
		`)
		if err == nil {
			defer rows.Close()
			log.Printf("Stats debug: Transaction dates analysis:")
			for rows.Next() {
				var date, total, dayPart, todayPart, monthPart, currentMonthPart, isToday, isThisMonth string
				if err := rows.Scan(&date, &total, &dayPart, &todayPart, &monthPart, &currentMonthPart, &isToday, &isThisMonth); err == nil {
					log.Printf("  Date: %s, Total: %s", date, total)
					log.Printf("    Day part: %s, Today part: %s, Is Today: %s", dayPart, todayPart, isToday)
					log.Printf("    Month part: %s, Current Month part: %s, Is This Month: %s", monthPart, currentMonthPart, isThisMonth)
				}
			}
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
