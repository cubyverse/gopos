package main

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"gopos/components"
	"gopos/config"
	"gopos/database"
	"gopos/handlers"
	"gopos/services"

	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
)

//go:embed static
var staticFiles embed.FS

var (
	version  = "development"
	commitID = "unknown"
)

func getConfigLocations() []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Warning: Could not get user home directory: %v", err)
		homeDir = ""
	}

	// Default locations to check for config file
	locations := []string{
		"config.yaml", // Current directory
	}

	if homeDir != "" {
		locations = append(locations, filepath.Join(homeDir, ".gopos", "config.yaml"))
	}

	// Add system-wide config location based on OS
	if os.PathSeparator == '/' { // Unix-like systems
		locations = append(locations, "/etc/gopos/config.yaml")
	} else { // Windows
		locations = append(locations, filepath.Join(os.Getenv("ProgramData"), "gopos", "config.yaml"))
	}

	return locations
}

func loadConfig() (*config.Config, error) {
	var lastErr error
	locations := getConfigLocations()

	for _, loc := range locations {
		data, err := os.ReadFile(loc)
		if err != nil {
			lastErr = err
			continue
		}

		var cfg config.Config
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			lastErr = err
			continue
		}

		absPath, err := filepath.Abs(loc)
		if err != nil {
			log.Printf("Loaded configuration from: %s (could not resolve to absolute path: %v)", loc, err)
		} else {
			log.Printf("Loaded configuration from: %s", absPath)
		}
		return &cfg, nil
	}

	return nil, fmt.Errorf("could not load config from any location, last error: %v", lastErr)
}

func main() {
	// Load configuration
	config, err := loadConfig()
	if err != nil {
		log.Fatal("Error loading config:", err)
	}

	// Initialize session store
	handlers.InitSessionStore(config)

	// Initialize email service
	services.InitEmailService(config)

	// Initialize database
	db, err := sql.Open("sqlite", config.Database.Path)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check if database file exists, if not it will be created automatically by SQLite
	// when we initialize the schema
	dbAbsPath, err := filepath.Abs(config.Database.Path)
	if err != nil {
		log.Printf("Using database at: %s (could not resolve to absolute path: %v)", config.Database.Path, err)
	} else {
		log.Printf("Using database at: %s", dbAbsPath)
	}
	if _, err := os.Stat(config.Database.Path); os.IsNotExist(err) {
		log.Printf("Database file does not exist, it will be created automatically")

		// Ensure the directory exists
		dbDir := filepath.Dir(config.Database.Path)
		if dbDir != "" && dbDir != "." {
			if err := os.MkdirAll(dbDir, 0755); err != nil {
				log.Fatalf("Failed to create database directory: %v", err)
			}
		}
	}

	// Initialize database schema
	if err := database.InitDB(db); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Create a new ServeMux
	mux := http.NewServeMux()

	// Serve embedded static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Middleware to inject database and version info into context
	withDB := func(handler http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), handlers.DbKey, db)
			ctx = context.WithValue(ctx, "version", version)
			ctx = context.WithValue(ctx, "commitID", commitID)
			handler.ServeHTTP(w, r.WithContext(ctx))
		}
	}

	// Define routes with their handlers
	routes := map[string]http.HandlerFunc{
		// Public routes
		"/":       withDB(handlers.HandleLogin(db)),
		"/login":  withDB(handlers.HandleLoginPost(db)),
		"/logout": withDB(handlers.HandleLogout),

		// Protected routes - require authentication
		"/dashboard":    withDB(handlers.RequireAuth(handlers.HandleDashboard(db))),
		"/transactions": withDB(handlers.RequireAuth(handlers.HandleTransactions(db))),

		// Admin routes
		"/users":           withDB(handlers.RequireAuth(handlers.RequireRole([]string{"admin"}, handlers.HandleUsers(db)))),
		"/users/new":       withDB(handlers.RequireAuth(handlers.RequireRole([]string{"admin"}, handlers.HandleNewUser(db)))),
		"/users/delete":    withDB(handlers.RequireAuth(handlers.RequireRole([]string{"admin"}, handlers.HandleDeleteUser(db)))),
		"/users/edit":      withDB(handlers.RequireAuth(handlers.RequireRole([]string{"admin"}, handlers.HandleEditUser(db)))),
		"/users/topup":     withDB(handlers.RequireAuth(handlers.RequireRole([]string{"admin", "cashier"}, handlers.HandleTopupUser(db)))),
		"/users/search":    withDB(handlers.RequireAuth(handlers.RequireRole([]string{"admin"}, handlers.HandleUserSearch(db)))),
		"/users/filter":    withDB(handlers.RequireAuth(handlers.RequireRole([]string{"admin"}, handlers.HandleUserFilter(db)))),
		"/audit":           withDB(handlers.RequireAuth(handlers.RequireRole([]string{"admin"}, handlers.HandleAuditTrail(db)))),
		"/stats":           withDB(handlers.RequireAuth(handlers.RequireRole([]string{"admin"}, handlers.HandleStats(db)))),
		"/products":        withDB(handlers.RequireAuth(handlers.RequireRole([]string{"admin"}, handlers.HandleProducts(db)))),
		"/products/new":    withDB(handlers.RequireAuth(handlers.RequireRole([]string{"admin"}, handlers.HandleNewProduct(db)))),
		"/products/edit":   withDB(handlers.RequireAuth(handlers.RequireRole([]string{"admin"}, handlers.HandleEditProduct(db)))),
		"/products/delete": withDB(handlers.RequireAuth(handlers.RequireRole([]string{"admin"}, handlers.HandleDeleteProduct(db)))),
		"/products/search": withDB(handlers.RequireAuth(handlers.RequireRole([]string{"admin"}, handlers.HandleProductSearch(db)))),
		"/products/filter": withDB(handlers.RequireAuth(handlers.RequireRole([]string{"admin"}, handlers.HandleProductFilter(db)))),

		// Cashier routes
		"/checkout":      withDB(handlers.RequireAuth(handlers.RequireRole([]string{"admin", "cashier"}, handlers.HandleCheckout(db)))),
		"/balance/topup": withDB(handlers.RequireAuth(handlers.RequireRole([]string{"admin", "cashier"}, handlers.HandleBalanceTopup(db)))),

		// API routes
		"/api/customers": withDB(handlers.RequireAuth(handlers.RequireRole([]string{"admin", "cashier"}, handlers.HandleCustomerLookup(db)))),
		"/api/products":  withDB(handlers.RequireAuth(handlers.RequireRole([]string{"admin", "cashier"}, handlers.HandleProductScan(db)))),
		"/api/checkout":  withDB(handlers.RequireAuth(handlers.RequireRole([]string{"admin", "cashier"}, handlers.HandleCompleteCheckout(db)))),
	}

	// Register all routes
	for path, handler := range routes {
		mux.HandleFunc(path, handler)
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Version: %s (Commit: %s)", components.Version, components.CommitID)
	log.Printf("Server starting on port %s...", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
