package harness

import (
	"fmt"
	"os"
	"strings"
)

func runAdmin(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: world-tool admin reset-password [flags]")
		return 2
	}
	switch args[0] {
	case "reset-password":
		return cmdAdminResetPassword(args[1:])
	case "revoke-sessions":
		return cmdAdminRevokeSessions(args[1:])
	default:
		fmt.Fprintln(os.Stderr, "unknown admin command")
		return 2
	}
}

func cmdAdminResetPassword(args []string) int {
	fs := flagSet("admin reset-password")
	authDB := fs.String("auth-db", envDefault("WORLD_HARNESS_AUTH_DB", "/app/data/auth.sqlite"), "auth sqlite path")
	username := fs.String("username", "", "username")
	password := fs.String("password", "", "new password")
	passwordFile := fs.String("password-file", "", "new password file")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if *username == "" {
		fmt.Fprintln(os.Stderr, "--username is required")
		return 2
	}
	raw := *password
	if *passwordFile != "" {
		b, err := os.ReadFile(*passwordFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		raw = strings.TrimSpace(string(b))
	}
	if raw == "" {
		fmt.Fprintln(os.Stderr, "--password or --password-file is required")
		return 2
	}
	store, err := openAuthStore(*authDB)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := store.resetPasswordByUsername(*username, raw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "password reset for %s\n", *username)
	return 0
}

func cmdAdminRevokeSessions(args []string) int {
	fs := flagSet("admin revoke-sessions")
	authDB := fs.String("auth-db", envDefault("WORLD_HARNESS_AUTH_DB", "/app/data/auth.sqlite"), "auth sqlite path")
	username := fs.String("username", "", "username")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if *username == "" {
		fmt.Fprintln(os.Stderr, "--username is required")
		return 2
	}
	store, err := openAuthStore(*authDB)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	var id string
	if err := store.db.QueryRow(`SELECT id FROM users WHERE username=?`, *username).Scan(&id); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := store.revokeUserSessions(id); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "sessions revoked for %s\n", *username)
	return 0
}
