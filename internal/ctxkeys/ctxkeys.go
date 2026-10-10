// Package ctxkeys defines typed keys for values stored in request contexts,
// shared between the HTTP middleware that writes them and the controllers that
// read them.
package ctxkeys

// Key is the type used for context keys to avoid collisions with other packages.
type Key string

// UserID is the context key under which the authenticated user's ID is stored.
const UserID Key = "userID"
