package harness

import (
	"context"
	"database/sql"
	"net/http"
)

type authStore struct {
	db *sql.DB
}

type authUser struct {
	ID          string
	Username    string
	DisplayName string
	Role        string
	Status      string
	LastLoginAt string
}

type authContextKey struct{}

func currentUser(r *http.Request) *authUser {
	u, _ := r.Context().Value(authContextKey{}).(*authUser)
	return u
}

func withUser(r *http.Request, u *authUser) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), authContextKey{}, u))
}
