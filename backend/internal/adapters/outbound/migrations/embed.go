// Package migrations embeds the SQL migrations so the binary ships them.
package migrations

import (
	"embed"
	"fmt"
	"sort"
	"strings"

	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/adapters/outbound/sqlite"
)

//go:embed *.sql
var fs embed.FS

// Load returns the migrations embedded in the binary, in version order.
func Load() ([]sqlite.Migration, error) {
	entries, err := fs.ReadDir(".")
	if err != nil {
		return nil, err
	}
	var out []sqlite.Migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		// Filenames like "0001_init.sql"
		parts := strings.SplitN(e.Name(), "_", 2)
		if len(parts) == 0 {
			continue
		}
		var version int
		if _, err := fmt.Sscanf(parts[0], "%d", &version); err != nil {
			continue
		}
		data, err := fs.ReadFile(e.Name())
		if err != nil {
			return nil, err
		}
		out = append(out, sqlite.Migration{
			Version: version,
			Name:    strings.TrimSuffix(e.Name(), ".sql"),
			SQL:     string(data),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}