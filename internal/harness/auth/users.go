package auth

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

func (s *Store) ListUsers() ([]map[string]any, error) {
	rows, err := s.db.Query(`SELECT u.id, u.username, u.display_name, u.role, u.status, COALESCE(u.last_login_at,''), COUNT(s.id)
FROM users u LEFT JOIN sessions s ON s.user_id=u.id AND s.revoked_at IS NULL AND s.expires_at > ?
GROUP BY u.id ORDER BY u.username`, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, username, display, role, status, last string
		var sessions int
		if err := rows.Scan(&id, &username, &display, &role, &status, &last, &sessions); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "username": username, "display_name": display, "role": role, "status": status, "last_login_at": last, "active_sessions": sessions})
	}
	return out, rows.Err()
}

func (s *Store) CreateUser(username, display, role, password string) error {
	if role != "admin" {
		role = "friend"
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.Exec(`INSERT INTO users (id, username, display_name, role, password_hash, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, 'active', ?, ?)`,
		"user_"+slugID(username), username, firstNonEmpty(display, username), role, string(hash), now, now)
	return err
}

func (s *Store) UpdateUser(id, role, status string) error {
	if role != "admin" {
		role = "friend"
	}
	if status != "disabled" {
		status = "active"
	}
	_, err := s.db.Exec(`UPDATE users SET role=?, status=?, updated_at=? WHERE id=?`, role, status, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

func (s *Store) ResetPassword(id, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE users SET password_hash=?, updated_at=? WHERE id=?`, string(hash), time.Now().UTC().Format(time.RFC3339), id)
	return err
}

func (s *Store) ResetPasswordByUsername(username, password string) error {
	var id string
	if err := s.db.QueryRow(`SELECT id FROM users WHERE username=?`, username).Scan(&id); err != nil {
		return err
	}
	return s.ResetPassword(id, password)
}

func (s *Store) FirstActiveAdminID() (string, error) {
	var id string
	err := s.db.QueryRow(`SELECT id FROM users WHERE role='admin' AND status='active' ORDER BY created_at, username LIMIT 1`).Scan(&id)
	return id, err
}
