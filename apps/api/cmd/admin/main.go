// Bootstrap an administrator explicitly; public signup never grants admin access.
package main

import (
	"context"
	"expertsacademy/blog/internal/domain"
	"expertsacademy/blog/internal/repository"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"net/mail"
	"os"
	"strings"
	"time"
)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run() error {
	email, name, password := os.Getenv("ADMIN_EMAIL"), os.Getenv("ADMIN_NAME"), os.Getenv("ADMIN_PASSWORD")
	a, e := mail.ParseAddress(email)
	if e != nil || a.Address != email || len(strings.TrimSpace(name)) < 2 || len(password) < 12 || len(password) > 72 {
		return fmt.Errorf("set ADMIN_EMAIL, ADMIN_NAME and ADMIN_PASSWORD (12–72 bytes)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, e := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if e != nil {
		return e
	}
	defer pool.Close()
	hash, e := bcrypt.GenerateFromPassword([]byte(password), 12)
	if e != nil {
		return e
	}
	_, e = repository.New(pool).CreateUser(ctx, domain.User{Email: email, Name: name, PasswordHash: string(hash), Role: "admin"})
	if e != nil {
		return e
	}
	fmt.Println("Administrator created.")
	return nil
}
