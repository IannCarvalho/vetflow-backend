package routes

import (
	"context"
	"net/http"
	"strings"

	"github.com/vetflow/veterinary-api/helpers"
)

// AuthMiddleware validates the JWT from the Authorization header and injects the
// authenticated user's ID into the request context.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"error":"invalid authorization header format"}`, http.StatusUnauthorized)
			return
		}

		// Validate the JWT and read the user ID from its claims.
		claims, err := helpers.ParseToken(parts[1])
		if err != nil || claims.UserID == "" {
			http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		// Add user ID to request context.
		// Key must match the one read by getUserIDFromContext in controllers/helpers.go.
		ctx := context.WithValue(r.Context(), "userID", claims.UserID) //nolint:staticcheck // string key matches existing reader
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
