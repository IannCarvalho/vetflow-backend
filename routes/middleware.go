package routes

import (
	"context"
	"net/http"
	"strings"
)

// AuthMiddleware validates token from Authorization header
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

		// For now, extract user ID from context (passed from OAuth or auth flow)
		// In production, validate JWT token properly
		userID := extractUserIDFromHeader(r)
		if userID == "" {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		// Add user ID to request context
		ctx := context.WithValue(r.Context(), "userID", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractUserIDFromHeader extracts user ID from request headers or token
// This is a simplified implementation - in production, use proper JWT validation
func extractUserIDFromHeader(r *http.Request) string {
	// Try to get from custom header
	if userID := r.Header.Get("X-User-ID"); userID != "" {
		return userID
	}

	// TODO: Implement proper JWT token parsing when JWT is implemented
	// For now, accept any token and expect X-User-ID header

	return ""
}
