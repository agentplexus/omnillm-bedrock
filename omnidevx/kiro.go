package omnidevx

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	core "github.com/plexusone/omnidevx-core"
	_ "modernc.org/sqlite" // pure-Go sqlite driver, registered as "sqlite"
)

const (
	DefaultConfidence        = 0.65
	DefaultCharsPerToken     = 4
	DefaultArchivedDirectory = ".kiro_sessions"
)

var source = core.Source{
	Provider: "aws",
	Product:  "kiro-cli",
}

// Config configures the Kiro collector.
type Config struct {
	// DBPath is the Kiro CLI SQLite database. Defaults to Kiro's OS-specific
	// local data path.
	DBPath string
	// SessionsDir is the optional archive directory created by kiro-usage-like
	// archivers. Defaults to ~/.kiro_sessions.
	SessionsDir string
	// CharsPerToken controls text-length token estimation. Defaults to 4.
	CharsPerToken int64
}

// Collector reads AWS Kiro local CLI usage.
type Collector struct {
	dbPath        string
	sessionsDir   string
	charsPerToken int64
}

var _ core.Collector = (*Collector)(nil)

// New returns a Collector for the given config.
func New(cfg Config) (*Collector, error) {
	dbPath := cfg.DBPath
	sessionsDir := cfg.SessionsDir
	if dbPath == "" || sessionsDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("kiro: resolve home directory: %w", err)
		}
		if dbPath == "" {
			dbPath = defaultDBPath(home)
		}
		if sessionsDir == "" {
			sessionsDir = filepath.Join(home, DefaultArchivedDirectory)
		}
	}
	cpt := cfg.CharsPerToken
	if cpt <= 0 {
		cpt = DefaultCharsPerToken
	}
	return &Collector{dbPath: dbPath, sessionsDir: sessionsDir, charsPerToken: cpt}, nil
}

// Source implements omnidevx.Collector.
func (c *Collector) Source() core.Source { return source }

// Collect implements omnidevx.Collector.
func (c *Collector) Collect(ctx context.Context, req core.CollectRequest) (*core.CollectionResult, error) {
	result := &core.CollectionResult{
		Source:      source,
		Subject:     req.Subject,
		Period:      req.Period,
		Events:      []core.Event{},
		CollectedAt: time.Now().UTC(),
	}

	seen := map[string]int64{}
	if err := c.collectArchives(ctx, req, result, seen); err != nil {
		return nil, err
	}
	if err := c.collectSQLite(ctx, req, result, seen); err != nil {
		return nil, err
	}
	for i := range result.Events {
		result.Events[i].Subject = req.Subject
	}
	return result, nil
}

func defaultDBPath(home string) string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "kiro-cli", "data.sqlite3")
	}
	return filepath.Join(home, ".local", "share", "kiro-cli", "data.sqlite3")
}

func provenance() core.Provenance {
	return core.Provenance{
		CollectionMode: core.ModeHistory,
		Confidence:     DefaultConfidence,
	}
}

func (c *Collector) collectSQLite(ctx context.Context, req core.CollectRequest, result *core.CollectionResult, seen map[string]int64) error {
	if _, err := os.Stat(c.dbPath); os.IsNotExist(err) {
		return nil
	}
	mtime := fileModUnixMS(c.dbPath)
	db, err := sql.Open("sqlite", "file:"+c.dbPath+"?mode=ro")
	if err != nil {
		return fmt.Errorf("kiro: open db %s: %w", c.dbPath, err)
	}
	defer db.Close() //nolint:errcheck // read-only handle

	if err := c.collectConversationsV2(ctx, db, req, result, seen); err != nil {
		return err
	}
	if err := c.collectConversations(ctx, db, req, result, seen, mtime); err != nil {
		return err
	}
	return nil
}

func (c *Collector) collectConversationsV2(ctx context.Context, db *sql.DB, req core.CollectRequest, result *core.CollectionResult, seen map[string]int64) error {
	rows, err := db.QueryContext(ctx, `SELECT conversation_id, key, created_at, updated_at, value FROM conversations_v2`)
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, core.Diagnostic{
			Severity: core.SeverityWarning,
			Message:  fmt.Sprintf("query conversations_v2: %v", err),
			Path:     c.dbPath,
		})
		return nil
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var cid, cwd, value string
		var created, updated int64
		if err := rows.Scan(&cid, &cwd, &created, &updated, &value); err != nil {
			result.Diagnostics = append(result.Diagnostics, core.Diagnostic{Severity: core.SeverityWarning, Message: fmt.Sprintf("scan conversations_v2: %v", err), Path: c.dbPath})
			continue
		}
		if seen[cid] >= updated && updated > 0 {
			continue
		}
		events, diags := c.parseSnapshot(snapshot{ConversationID: cid, CWD: cwd, CreatedAt: created, UpdatedAt: updated, Value: json.RawMessage(value)}, req, c.dbPath)
		result.Events = append(result.Events, events...)
		result.Diagnostics = append(result.Diagnostics, diags...)
		seen[cid] = updated
	}
	return rows.Err()
}

func (c *Collector) collectConversations(ctx context.Context, db *sql.DB, req core.CollectRequest, result *core.CollectionResult, seen map[string]int64, fallbackMS int64) error {
	rows, err := db.QueryContext(ctx, `SELECT key, value FROM conversations`)
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, core.Diagnostic{
			Severity: core.SeverityWarning,
			Message:  fmt.Sprintf("query conversations: %v", err),
			Path:     c.dbPath,
		})
		return nil
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var cwd, value string
		if err := rows.Scan(&cwd, &value); err != nil {
			result.Diagnostics = append(result.Diagnostics, core.Diagnostic{Severity: core.SeverityWarning, Message: fmt.Sprintf("scan conversations: %v", err), Path: c.dbPath})
			continue
		}
		var conv conversation
		if err := json.Unmarshal([]byte(value), &conv); err != nil || conv.ConversationID == "" {
			result.Diagnostics = append(result.Diagnostics, core.Diagnostic{Severity: core.SeverityWarning, Message: fmt.Sprintf("parse conversations value: %v", err), Path: c.dbPath})
			continue
		}
		snap := snapshot{ConversationID: conv.ConversationID, CWD: cwd, CreatedAt: fallbackMS, UpdatedAt: fallbackMS, Value: json.RawMessage(value)}
		created, updated := conversationBounds(conv, fallbackMS)
		snap.CreatedAt = created
		snap.UpdatedAt = updated
		if seen[snap.ConversationID] >= snap.UpdatedAt && snap.UpdatedAt > 0 {
			continue
		}
		events, diags := c.parseSnapshot(snap, req, c.dbPath)
		result.Events = append(result.Events, events...)
		result.Diagnostics = append(result.Diagnostics, diags...)
		seen[snap.ConversationID] = snap.UpdatedAt
	}
	return rows.Err()
}

func fileModUnixMS(path string) int64 {
	st, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return st.ModTime().UnixMilli()
}

func msTime(ms int64) time.Time {
	if ms <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}
