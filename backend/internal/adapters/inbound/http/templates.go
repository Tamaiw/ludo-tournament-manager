package httpinbound

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/gorilla/csrf"

	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/domain"
)

//go:embed templates/**/*.html
var templatesFS embed.FS

// PageData is the common context passed to every page template.
type PageData struct {
	User            *domain.User
	Flash           *Flash
	CSRFField       template.HTML
	Tournament      *domain.Tournament
	Tournaments     []domain.Tournament
	Managed         []domain.Tournament
	Playing         []domain.Tournament
	Bracket         *domain.Bracket
	Matches         []domain.Match
	Participants    map[domain.MatchID][]domain.MatchParticipant
	Roster          []domain.TournamentPlayer
	Users           []domain.User
	SpectatorTokens []domain.SpectatorToken
	AuditEntries    []domain.AuditLogEntry
	Error           string
	Token           string
	TokenURL        string
	InviterName     string
	Form            map[string]string
}

// Flash carries a one-shot message between requests via session storage.
type Flash struct {
	Level   string // info, success, warning, error
	Message string
}

var (
	tmplOnce  sync.Once
	tmplErr   error
	layouts   *template.Template
	fragments = map[string]*template.Template{}
	pages     = map[string]*template.Template{}
)

func loadTemplatesOnce() error {
	tmplOnce.Do(func() {
		funcs := template.FuncMap{
			"fmtTime":    func(t time.Time) string { return t.Format(time.RFC3339) },
			"statusClass": func(s domain.TournamentStatus) string { return "status-" + string(s) },
		}
		// Parse the layout/header fragments first.
		layouts = template.New("").Funcs(funcs)
		entries, err := templatesFS.ReadDir("templates/layouts")
		if err != nil {
			tmplErr = fmt.Errorf("read layouts: %w", err)
			return
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".html") {
				continue
			}
			data, err := templatesFS.ReadFile("templates/layouts/" + e.Name())
			if err != nil {
				tmplErr = err
				return
			}
			if _, err := layouts.New("layouts/" + e.Name()).Parse(string(data)); err != nil {
				tmplErr = fmt.Errorf("parse layouts/%s: %w", e.Name(), err)
				return
			}
		}
		// Parse each page into its own cloned template set.
		pageEntries, err := templatesFS.ReadDir("templates/pages")
		if err != nil {
			tmplErr = fmt.Errorf("read pages: %w", err)
			return
		}
		for _, e := range pageEntries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".html") {
				continue
			}
			data, err := templatesFS.ReadFile("templates/pages/" + e.Name())
			if err != nil {
				tmplErr = err
				return
			}
			clone, err := layouts.Clone()
			if err != nil {
				tmplErr = err
				return
			}
			name := "pages/" + e.Name()
			if _, err := clone.New(name).Parse(string(data)); err != nil {
				tmplErr = fmt.Errorf("parse pages/%s: %w", e.Name(), err)
				return
			}
			pages[name] = clone
		}
		// Parse each fragment into its own set too.
		err = fs.WalkDir(templatesFS, "templates/fragments", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".html") {
				return nil
			}
			data, err := templatesFS.ReadFile(path)
			if err != nil {
				return err
			}
			clone, err := layouts.Clone()
			if err != nil {
				return err
			}
			name := strings.TrimPrefix(path, "templates/")
			if _, err := clone.New(name).Parse(string(data)); err != nil {
				return fmt.Errorf("parse %s: %w", path, err)
			}
			fragments[name] = clone
			return nil
		})
		if err != nil {
			tmplErr = err
			return
		}
	})
	return tmplErr
}

// render executes the page template by name (e.g. "pages/sign_in").
func render(w http.ResponseWriter, name string, data PageData) {
	if err := loadTemplatesOnce(); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if data.Form == nil {
		data.Form = map[string]string{}
	}
	key := name
	if !strings.HasSuffix(key, ".html") {
		key = name + ".html"
	}
	pg, ok := pages[key]
	if !ok {
		http.Error(w, "template not found: "+key, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pg.ExecuteTemplate(w, "layouts/base.html", data); err != nil {
		http.Error(w, "render error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// renderFragment executes a fragment template (e.g. "fragments/bracket").
func renderFragment(w http.ResponseWriter, name string, data PageData) {
	if err := loadTemplatesOnce(); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if data.Form == nil {
		data.Form = map[string]string{}
	}
	key := name
	if !strings.HasSuffix(key, ".html") {
		key = name + ".html"
	}
	pg, ok := fragments[key]
	if !ok {
		http.Error(w, "template not found: "+key, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pg.ExecuteTemplate(w, "layouts/base.html", data); err != nil {
		http.Error(w, "render error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// requireSessionManager retrieves the scs session manager from a context or panics.
func requireSessionManager(r *http.Request) *scs.SessionManager {
	sm := SessionManagerFromContext(r.Context())
	if sm == nil {
		panic("session manager not on context")
	}
	return sm
}

// renderCSRF returns the CSRF field HTML for inclusion in templates.
func renderCSRF(r *http.Request) template.HTML {
	return csrf.TemplateField(r)
}

// putFlash stores a one-shot flash message in the session.
func putFlash(sm *scs.SessionManager, r *http.Request, level, message string) {
	sm.Put(r.Context(), "flash", Flash{Level: level, Message: message})
}

// popFlash retrieves and removes any pending flash message.
func popFlash(sm *scs.SessionManager, r *http.Request) *Flash {
	v := sm.Pop(r.Context(), "flash")
	if v == nil {
		return nil
	}
	if f, ok := v.(Flash); ok {
		return &f
	}
	return nil
}