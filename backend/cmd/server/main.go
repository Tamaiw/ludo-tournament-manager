// Command server is the Ludo Tournament Manager HTTP server.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	httpinbound "github.com/Tamaiw/ludo-tournament-manager/backend/internal/adapters/inbound/http"
	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/adapters/inbound/ws"
	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/adapters/outbound/argon2"
	migrationspkg "github.com/Tamaiw/ludo-tournament-manager/backend/internal/adapters/outbound/migrations"
	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/adapters/outbound/smtp"
	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/adapters/outbound/sqlite"
	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/ports"
	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/services"
)

func main() {
	cfg := loadConfig()

	migs, err := migrationspkg.Load()
	if err != nil {
		log.Fatalf("load migrations: %v", err)
	}

	ctx := context.Background()
	db, err := sqlite.Open(ctx, cfg.DBPath, migs)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewRepos(db)
	hasher := argon2.New()
	email := smtp.NewSender(smtp.Config{
		Host:      cfg.SMTPHost,
		Port:      cfg.SMTPPort,
		Username:  cfg.SMTPUsername,
		Password:  cfg.SMTPPassword,
		From:      cfg.SMTPFrom,
		PublicURL: cfg.PublicURL,
	})
	broadcaster := ws.NewHub()
	clock := ports.RealClock{}

	csrfKey := []byte(getEnv("SESSION_KEY", "01234567890123456789012345678901"))
	if len(csrfKey) < 32 {
		log.Fatalf("SESSION_KEY must be at least 32 bytes (got %d)", len(csrfKey))
	}

	// Build every service.
	svc := services.New(services.BuildParams{
		Users:           repo.Users,
		Hasher:          hasher,
		Clock:           clock,
		Tournaments:     repo.Tournaments,
		TManagers:       repo.TournamentManagers,
		TPlayers:        repo.TournamentPlayers,
		AuthTokens:      repo.AuthTokens,
		SpectatorTokens: repo.TournamentSpectatorTokens,
		Matches:         repo.Matches,
		MatchParts:      repo.MatchParticipants,
		AuditLog:        repo.AuditLog,
		Email:           email,
		Broadcaster:     broadcaster,
		PublicURL:       cfg.PublicURL,
	})

	srv := httpinbound.New(httpinbound.Config{
		Users:              repo.Users,
		Tournaments:        repo.Tournaments,
		TournamentManagers: repo.TournamentManagers,
		TournamentPlayers:  repo.TournamentPlayers,
		SpectatorTokens:    repo.TournamentSpectatorTokens,
		Matches:            repo.Matches,
		AuditLog:           repo.AuditLog,
		AuthTokens:         repo.AuthTokens,
		Email:              email,
		Broadcaster:        broadcaster,
		Clock:              clock,
		Hasher:             hasher,
		Tx:                 repo.Tx,
		SessionStore:       repo.Sessions,
		CSRFKey:            csrfKey,
		CookieSecure:       cfg.CookieSecure,
		PublicURL:          cfg.PublicURL,
		Services:           svc,
	})

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("listening on :%s", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
}

type config struct {
	AppEnv       string
	DBPath       string
	Port         string
	CookieSecure bool
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
	PublicURL    string
}

func loadConfig() config {
	port := getEnv("PORT", "8080")
	return config{
		AppEnv:       getEnv("APP_ENV", "development"),
		DBPath:       getEnv("DB_PATH", "./data/ludo.db"),
		Port:         port,
		CookieSecure: getEnv("COOKIE_SECURE", "false") == "true" || getEnv("APP_ENV", "") == "production",
		SMTPHost:     getEnv("SMTP_HOST", ""),
		SMTPPort:     getEnvInt("SMTP_PORT", 1025),
		SMTPUsername: getEnv("SMTP_USERNAME", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:     getEnv("SMTP_FROM", ""),
		PublicURL:    getEnv("PUBLIC_URL", "http://localhost:"+port),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
