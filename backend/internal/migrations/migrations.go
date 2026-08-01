// Package migrations holds versioned, hand-authored SQL files for schema
// changes that GORM's AutoMigrate cannot safely perform on its own (column
// renames, type changes, data backfills). AutoMigrate keeps handling purely
// additive changes (new tables/columns) automatically on every boot, as it
// always has — this package is only for the rest, applied deliberately by an
// admin from the web UI (dry-run first, then apply).
package migrations

import (
	"embed"
	"sort"
	"strconv"
	"strings"
)

//go:embed sql/*.sql
var sqlFiles embed.FS

// Migration is one versioned upgrade step: applying it takes the schema from
// Version-1 to Version.
type Migration struct {
	Version     int
	Filename    string
	Description string
	SQL         string
}

// Statements splits the migration's SQL into individual statements. Each is
// run separately (rather than as one multi-statement Exec) since the
// Postgres extended query protocol pgx uses by default cannot execute more
// than one command per prepared statement.
func (m Migration) Statements() []string {
	var out []string
	for _, stmt := range strings.Split(m.SQL, ";") {
		if t := strings.TrimSpace(stmt); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// All returns every embedded migration, sorted by version ascending.
func All() []Migration {
	entries, err := sqlFiles.ReadDir("sql")
	if err != nil {
		return nil
	}
	var out []Migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		version, description, ok := parseFilename(e.Name())
		if !ok {
			continue
		}
		content, err := sqlFiles.ReadFile("sql/" + e.Name())
		if err != nil {
			continue
		}
		out = append(out, Migration{Version: version, Filename: e.Name(), Description: description, SQL: string(content)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out
}

// parseFilename extracts the version and a human description from
// "0002_add_demo_column.sql" -> (2, "add demo column", true).
func parseFilename(name string) (version int, description string, ok bool) {
	base := strings.TrimSuffix(name, ".sql")
	parts := strings.SplitN(base, "_", 2)
	if len(parts) != 2 {
		return 0, "", false
	}
	v, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", false
	}
	return v, strings.ReplaceAll(parts[1], "_", " "), true
}

// Pending returns migrations with a version strictly greater than current.
func Pending(current int) []Migration {
	var out []Migration
	for _, m := range All() {
		if m.Version > current {
			out = append(out, m)
		}
	}
	return out
}

// ByVersion finds a single migration by its target version.
func ByVersion(version int) (Migration, bool) {
	for _, m := range All() {
		if m.Version == version {
			return m, true
		}
	}
	return Migration{}, false
}
