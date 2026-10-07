// Command server is the Ludo Tournament Manager HTTP server.
//
// In T01 it only exposes GET /healthz, opens SQLite, applies migrations, and
// exits cleanly on signal. Subsequent tickets wire the rest of the surface
// (sign-in, tournaments, brackets, WebSocket, etc.).
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"

	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/adapters/outbound/argon2"
	migrationspkg "github.com/Tamaiw/ludo-tournament-manager/backend/internal/adapters/outbound/migrations"
	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/adapters/outbound/smtp"
	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/adapters/outbound/sqlite"
	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/adapters/inbound/ws"
	httpinbound "github.com/Tamaiw/ludo-tournament-manager/backend/internal/adapters/inbound/http"
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
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		Username: cfg.SMTPUsername,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
		PublicURL: cfg.PublicURL,
	})
	broadcaster := ws.NewHub()

	// Session manager (scs) is wired against the SQLite session store.
	sessionManager := scs.New()
	sessionManager.Lifetime = 7 * 24 * time.Hour
	sessionManager.IdleTimeout = 24 * time.Hour
	sessionManager.Cookie.Name = "__Host-id"
	sessionManager.Cookie.HttpOnly = true
	sessionManager.Cookie.SameSite = http.SameSiteStrictMode
	sessionManager.Cookie.Secure = cfg.CookieSecure
	sessionManager.Cookie.Path = "/"
	sessionManager.HashTokenInStore = true
	sessionManager.Store = httpinbound.NewSCSStoreAdapter(repo.Sessions)

	r := chi.NewRouter()
	r.Use(httpinbound.RequestID)
	r.Use(httpinbound.RealIP)
	r.Use(httpinbound.Logger)
	r.Use(httpinbound.Recoverer)
	r.Use(sessionManager.LoadAndSave)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := db.PingContext(r.Context()); err != nil {
			http.Error(w, "unhealthy", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}

	_ = repo
	_ = hasher
	_ = email
	_ = broadcaster
}

type config struct {
	AppEnv        string
	DBPath        string
	Port          string
	CookieSecure  bool
	SMTPHost      string
	SMTPPort      int
	SMTPUsername  string
	SMTPPassword  string
	SMTPFrom      string
	PublicURL     string
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

// Mark a few variables that lint wants used (referenced from main indirectly).
var _ = sql.ErrNoRows