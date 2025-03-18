package handlers

import (
	"database/sql"
	"gopos/components"
	"gopos/config"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/sessions"
)

var store *sessions.CookieStore

const sessionName = "pos-session" // Consistent session name

// InitSessionStore initializes the session store with the key from config
func InitSessionStore(cfg *config.Config) {
	// Get session key from config
	sessionKey := cfg.Session.Key
	if sessionKey == "" {
		log.Fatal("Session key must be configured in config file")
	}
	store = sessions.NewCookieStore([]byte(sessionKey))

	// Configure secure cookie settings
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   os.Getenv("ENV") == "production", // Only use HTTPS in production
		SameSite: http.SameSiteStrictMode,
	}
}

// Custom type for context keys to avoid collisions
type contextKey string

const (
	userIDKey      contextKey = "userID"
	DbKey          contextKey = "db"
	contextUserKey contextKey = "user" // Renamed to avoid conflict
)

// HandleLogin renders the login page
func HandleLogin(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if user is already authenticated
		session, err := store.Get(r, sessionName)
		if err != nil {
			http.Error(w, "Session error", http.StatusInternalServerError)
			return
		}

		// If user is already authenticated, redirect to dashboard
		if auth, ok := session.Values["authenticated"].(bool); ok && auth {
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}

		data := components.LoginData{
			CSRFToken: generateCSRFToken(),
			Error:     "",
		}
		components.Login(data).Render(r.Context(), w)
	}
}

// HandleLoginPost processes the login form
func HandleLoginPost(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if user is already logged in
		session, _ := store.Get(r, sessionName)
		if auth, ok := session.Values["authenticated"].(bool); ok && auth {
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}

		if err := r.ParseForm(); err != nil {
			log.Printf("Login error: Form parsing failed: %v", err)
			data := components.LoginData{
				CSRFToken: generateCSRFToken(),
				Error:     "Ungültiges Formular",
			}
			components.Login(data).Render(r.Context(), w)
			return
		}

		token := r.FormValue("csrf_token")
		if !verifyCSRFToken(token) {
			log.Printf("Login error: Invalid CSRF token")
			data := components.LoginData{
				CSRFToken: generateCSRFToken(),
				Error:     "Ungültiger CSRF-Token",
			}
			components.Login(data).Render(r.Context(), w)
			return
		}

		cardNumber := r.FormValue("card_number")
		if cardNumber == "" {
			log.Printf("Login error: Empty card number")
			data := components.LoginData{
				CSRFToken: generateCSRFToken(),
				Error:     "Kartennummer erforderlich",
			}
			components.Login(data).Render(r.Context(), w)
			return
		}

		var user struct {
			ID    int
			Role  string
			Name  string
			Email sql.NullString
		}

		log.Printf("Login attempt with card number: %s", cardNumber)
		err := db.QueryRow("SELECT id, role, name, email FROM users WHERE card_number = ?", cardNumber).Scan(&user.ID, &user.Role, &user.Name, &user.Email)
		if err == sql.ErrNoRows {
			log.Printf("Login failed: Invalid card number: %s", cardNumber)
			data := components.LoginData{
				CSRFToken: generateCSRFToken(),
				Error:     "Ungültige Kartennummer",
			}
			components.Login(data).Render(r.Context(), w)
			return
		} else if err != nil {
			log.Printf("Login database error: %v", err)
			data := components.LoginData{
				CSRFToken: generateCSRFToken(),
				Error:     "Datenbankfehler",
			}
			components.Login(data).Render(r.Context(), w)
			return
		}

		log.Printf("Login successful: User %s (ID: %d) with role %s", user.Name, user.ID, user.Role)

		// Create new session
		session.Values["authenticated"] = true
		session.Values["user_id"] = user.ID
		session.Values["role"] = user.Role
		session.Values["name"] = user.Name
		session.Values["email"] = user.Email.String
		if err := session.Save(r, w); err != nil {
			log.Printf("Session error: %v", err)
			data := components.LoginData{
				CSRFToken: generateCSRFToken(),
				Error:     "Sitzungsfehler",
			}
			components.Login(data).Render(r.Context(), w)
			return
		}

		// Log the login
		now := time.Now()
		_, err = db.Exec(`
			INSERT INTO audit_log (user_id, action, details, created_at)
			VALUES (?, ?, ?, ?)
		`, user.ID, "login", "Benutzer eingeloggt: "+user.Name, now)

		if err != nil {
			log.Printf("Audit log error: %v", err)
		}

		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	}
}

// HandleLogout processes the logout request
func HandleLogout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)

	// Get user info for logging before clearing the session
	userID, ok := session.Values["user_id"].(int)
	userName, _ := session.Values["name"].(string)

	// Clear session
	session.Values["authenticated"] = false
	delete(session.Values, "user_id")
	delete(session.Values, "role")
	delete(session.Values, "name")
	session.Save(r, w)

	// Log the logout if we have the user info
	if ok {
		db, ok := r.Context().Value(DbKey).(*sql.DB)
		if ok {
			now := time.Now()
			_, err := db.Exec(`
				INSERT INTO audit_log (user_id, action, details, created_at)
				VALUES (?, ?, ?, ?)
			`, userID, "logout", "Benutzer ausgeloggt: "+userName, now)

			if err != nil {
				log.Printf("Audit log error: %v", err)
			}
		}
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// RequireRole middleware checks if user has required role
func RequireRole(roles []string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, sessionName)
		userRole, ok := session.Values["role"].(string)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		for _, role := range roles {
			if role == userRole {
				next.ServeHTTP(w, r)
				return
			}
		}

		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
}
