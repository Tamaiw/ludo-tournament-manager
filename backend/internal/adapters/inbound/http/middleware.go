// Package httpinbound is the inbound HTTP adapter for the Ludo Tournament Manager.
package httpinbound

import (
	"bufio"
	"context"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"

	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/adapters/outbound/sqlite"
)

// RequestID attaches an X-Request-ID header to every response (and the
// request context). The value is reused from the incoming header if present.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = newID()
		}
		w.Header().Set("X-Request-Id", id)
		ctx := context.WithValue(r.Context(), reqIDKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RealIP extracts the request's IP from r.RemoteAddr or X-Forwarded-For.
func RealIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.Header.Get("X-Forwarded-For")
		if ip != "" {
			if i := indexComma(ip); i >= 0 {
				ip = ip[:i]
			}
		} else {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err == nil {
				ip = host
			} else {
				ip = r.RemoteAddr
			}
		}
		ctx := context.WithValue(r.Context(), realIPKey{}, ip)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Logger logs every request with method, path, status, duration, and IP.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		log.Printf("%s %s %d %s from %s", r.Method, r.URL.Path, rw.status, time.Since(start), IP(r.Context()))
	})
}

// Recoverer catches panics in downstream handlers and returns 500.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic: %v", rec)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// SCSStoreAdapter adapts our sqlite.SessionStore to scs.Store. scs expects
// the cookie value (post-HashTokenInStore) to be the key in the store.
type SCSStoreAdapter struct {
	store *sqlite.SessionStore
}

// NewSCSStoreAdapter wraps a *sqlite.SessionStore.
func NewSCSStoreAdapter(store *sqlite.SessionStore) *SCSStoreAdapter {
	return &SCSStoreAdapter{store: store}
}

// Delete implements scs.Store.
func (a *SCSStoreAdapter) Delete(token string) error {
	return a.store.Delete(context.Background(), token)
}

// Find implements scs.Store.
func (a *SCSStoreAdapter) Find(token string) ([]byte, bool, error) {
	return a.store.Find(context.Background(), token)
}

// Commit implements scs.Store.
func (a *SCSStoreAdapter) Commit(token string, data []byte, expiry time.Time) error {
	return a.store.Commit(context.Background(), token, data, expiry.Unix())
}

// All implements scs.Store (used by scs's Iteration channel).
func (a *SCSStoreAdapter) All() (map[string]time.Time, error) {
	raw, err := a.store.All(context.Background())
	if err != nil {
		return nil, err
	}
	out := map[string]time.Time{}
	for k, v := range raw {
		out[k] = time.Unix(v, 0)
	}
	return out, nil
}

// Wrap scs session manager in case we want a helper for tests later.
type sessionManagerKey struct{}

// WithSessionManager stores the scs.SessionManager on the request context.
func WithSessionManager(r *http.Request, sm *scs.SessionManager) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), sessionManagerKey{}, sm))
}

// FromContext returns the session manager from the request context, if any.
func SessionManagerFromContext(ctx context.Context) *scs.SessionManager {
	if sm, ok := ctx.Value(sessionManagerKey{}).(*scs.SessionManager); ok {
		return sm
	}
	return nil
}

// ---- helpers ----

type reqIDKey struct{}
type realIPKey struct{}

func IP(ctx context.Context) string {
	if ip, ok := ctx.Value(realIPKey{}).(string); ok {
		return ip
	}
	return ""
}

// RequestIDFromContext returns the request ID stamped on the context.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(reqIDKey{}).(string); ok {
		return id
	}
	return ""
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := r.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func newID() string {
	return time.Now().UTC().Format("20060102150405.000000")
}

func indexComma(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			return i
		}
	}
	return -1
}

// scsStoreAdapter is a thin convenience wrapper used by main.go.
func scsStoreAdapter(store *sqlite.SessionStore) scs.Store {
	return NewSCSStoreAdapter(store)
}
