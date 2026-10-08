// Command seed bootstraps the first manager (or any user) of the Ludo Tournament Manager.
// Usage:
//
//	docker compose run --rm seed --email <email> --password <password> --name <name>
//
// If the user already exists the command exits with a clear error.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/adapters/outbound/argon2"
	migrationspkg "github.com/Tamaiw/ludo-tournament-manager/backend/internal/adapters/outbound/migrations"
	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/adapters/outbound/sqlite"
	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/domain"
	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/ports"
	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/services"
)

func main() {
	email := flag.String("email", "", "Email of the user to create")
	password := flag.String("password", "", "Initial password (will be hashed)")
	displayName := flag.String("name", "", "Display name (defaults to email local-part)")
	flag.Parse()

	if *email == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "Usage: seed --email <email> --password <password> [--name <name>]")
		os.Exit(2)
	}

	cfg := loadEnv()

	migs, err := migrationspkg.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load migrations: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	db, err := sqlite.Open(ctx, cfg.DBPath, migs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	repo := sqlite.NewRepos(db)

	name := *displayName
	if name == "" {
		name = strings.SplitN(*email, "@", 2)[0]
	}

	svc := services.CreateUser{
		Users:  repo.Users,
		Hasher: argon2.New(),
		Clock:  ports.RealClock{},
	}

	u, err := svc.Handle(ctx, services.CreateUserCmd{
		Email:    *email,
		Name:     name,
		Password: *password,
	})
	if err != nil {
		if err == domain.ErrUserExists {
			fmt.Fprintf(os.Stderr, "user already exists: %s\n", *email)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "create user: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("first manager created: id=%s email=%s name=%s\n", u.ID, u.Email, u.Name)
}

type config struct {
	DBPath string
}

func loadEnv() config {
	cfg := config{DBPath: os.Getenv("DB_PATH")}
	if cfg.DBPath == "" {
		cfg.DBPath = "./data/ludo.db"
	}
	return cfg
}
