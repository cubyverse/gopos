package handlers

import (
	"context"
	"gopos/components"
	"net/http"
)

// Add version information to all requests
func WithVersion(version, commitID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), "version", version)
			ctx = context.WithValue(ctx, "commitID", commitID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth middleware checks if user is authenticated
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, sessionName)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		auth, ok := session.Values["authenticated"].(bool)
		if !ok || !auth {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// Get user information from session
		userID, ok := session.Values["user_id"].(int)
		if !ok {
			http.Redirect(w, r, "/", http.StatusSeeOther)
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

		// Create user object
		user := components.User{
			ID:   userID,
			Name: userName,
			Role: userRole,
		}

		// Get version info from context
		version := r.Context().Value("version").(string)
		commitID := r.Context().Value("commitID").(string)

		// Create new context with user and version info
		ctx := context.WithValue(r.Context(), "version", version)
		ctx = context.WithValue(ctx, "commitID", commitID)
		ctx = context.WithValue(ctx, contextUserKey, user)

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
