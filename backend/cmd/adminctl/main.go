// Command adminctl provisions operator access without seed data.
//
// Usage (DB_DSN env required; password via env so it never lands in history):
//
//	DB_DSN=... ADMINCTL_PASSWORD='...' adminctl create-admin --email ops@example.com
//	DB_DSN=... adminctl set-role --email user@example.com --role ADMIN
//	DB_DSN=... adminctl set-active --email user@example.com --active=false
//
// create-admin records age_confirmed_at because the operator attests the owner
// is 14+; it bypasses REGISTRATION_INVITE_CODE by design (operator action).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/repository"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: %s <create-admin|set-role|set-active> [flags]", os.Args[0])
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := repository.OpenDB(requiredEnv("DB_DSN"))
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()
	repos := repository.New(db, nil)

	var cmdErr error
	switch os.Args[1] {
	case "create-admin":
		cmdErr = createAdmin(ctx, repos, os.Args[2:])
	case "set-role":
		cmdErr = setRole(ctx, repos, os.Args[2:])
	case "set-active":
		cmdErr = setActive(ctx, repos, os.Args[2:])
	default:
		log.Fatalf("unknown command %q", os.Args[1])
	}
	if cmdErr != nil {
		log.Fatalf("%s: %v", os.Args[1], cmdErr)
	}
}

func requiredEnv(key string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		log.Fatalf("%s is required", key)
	}
	return v
}

func createAdmin(ctx context.Context, repos *repository.Repos, args []string) error {
	fs := flag.NewFlagSet("create-admin", flag.ContinueOnError)
	email := fs.String("email", "", "operator email (required)")
	display := fs.String("display-name", "Operator", "display name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	mail := strings.TrimSpace(strings.ToLower(*email))
	password := os.Getenv("ADMINCTL_PASSWORD")
	if !strings.Contains(mail, "@") {
		return errors.New("--email must be a valid email")
	}
	if len(password) < 12 {
		return errors.New("ADMINCTL_PASSWORD must be at least 12 characters")
	}
	if _, err := repos.Users.GetByEmail(ctx, mail); err == nil {
		return errors.New("email already registered (use set-role instead)")
	} else if !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u, err := repos.Users.CreateWithProfile(ctx, mail, string(hash), domain.RoleAdmin, *display, nil, true)
	if err != nil {
		return err
	}
	if err := repos.Audit.Record(ctx, nil, "ADMIN_CREATE", "user", u.ID, `{"via":"adminctl"}`, "adminctl"); err != nil {
		return fmt.Errorf("admin created but audit write failed: %w", err)
	}
	fmt.Printf("created admin %s (%s)\n", u.ID, u.Email)
	return nil
}

func setRole(ctx context.Context, repos *repository.Repos, args []string) error {
	fs := flag.NewFlagSet("set-role", flag.ContinueOnError)
	email := fs.String("email", "", "user email (required)")
	role := fs.String("role", "", "ADMIN|PARTNER|USER (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	mail := strings.TrimSpace(strings.ToLower(*email))
	r := strings.ToUpper(strings.TrimSpace(*role))
	if r != domain.RoleAdmin && r != domain.RolePartner && r != domain.RoleUser {
		return errors.New("--role must be ADMIN|PARTNER|USER")
	}
	u, err := repos.Users.GetByEmail(ctx, mail)
	if err != nil {
		return err
	}
	if err := repos.Users.SetRole(ctx, u.ID, r); err != nil {
		return err
	}
	if err := repos.Audit.Record(ctx, nil, "USER_SET_ROLE", "user", u.ID, `{"role":"`+r+`","via":"adminctl"}`, "adminctl"); err != nil {
		return fmt.Errorf("role set but audit write failed: %w", err)
	}
	fmt.Printf("set role %s -> %s\n", u.Email, r)
	return nil
}

func setActive(ctx context.Context, repos *repository.Repos, args []string) error {
	fs := flag.NewFlagSet("set-active", flag.ContinueOnError)
	email := fs.String("email", "", "user email (required)")
	active := fs.Bool("active", true, "true to unlock, false to lock")
	if err := fs.Parse(args); err != nil {
		return err
	}
	u, err := repos.Users.GetByEmail(ctx, strings.TrimSpace(strings.ToLower(*email)))
	if err != nil {
		return err
	}
	if err := repos.Users.SetActive(ctx, u.ID, *active); err != nil {
		return err
	}
	if err := repos.Audit.Record(ctx, nil, "USER_SET_ACTIVE", "user", u.ID, `{"via":"adminctl"}`, "adminctl"); err != nil {
		return fmt.Errorf("active flag set but audit write failed: %w", err)
	}
	fmt.Printf("set active %s -> %v\n", u.Email, *active)
	return nil
}
