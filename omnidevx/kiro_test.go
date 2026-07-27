package omnidevx

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	core "github.com/plexusone/omnidevx-core"
)

func TestSourceDescriptor(t *testing.T) {
	c, err := New(Config{DBPath: filepath.Join(t.TempDir(), "missing.sqlite3")})
	if err != nil {
		t.Fatal(err)
	}
	want := core.Source{Provider: "aws", Product: "kiro-cli"}
	if got := c.Source(); got != want {
		t.Errorf("Source(): got %+v, want %+v", got, want)
	}
}

func TestCollectEmpty(t *testing.T) {
	c, err := New(Config{
		DBPath:      filepath.Join(t.TempDir(), "missing.sqlite3"),
		SessionsDir: filepath.Join(t.TempDir(), "missing-sessions"),
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := c.Collect(context.Background(), core.CollectRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Events) != 0 || len(result.Diagnostics) != 0 {
		t.Errorf("empty collect: got %d events and %d diagnostics", len(result.Events), len(result.Diagnostics))
	}
}

func TestCollectSQLitePairSchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "data.sqlite3")
	createKiroDB(t, dbPath)

	c, err := New(Config{DBPath: dbPath, SessionsDir: filepath.Join(t.TempDir(), "archives")})
	if err != nil {
		t.Fatal(err)
	}
	result, err := c.Collect(context.Background(), core.CollectRequest{
		Subject: core.SubjectRef{PersonID: "person:test"},
	})
	if err != nil {
		t.Fatal(err)
	}

	counts := countByType(result.Events)
	if counts[core.EventSessionStarted] != 1 {
		t.Errorf("session starts: got %d, want 1", counts[core.EventSessionStarted])
	}
	if counts[core.EventSessionEnded] != 1 {
		t.Errorf("session ends: got %d, want 1", counts[core.EventSessionEnded])
	}
	if counts[core.EventUsageRecorded] != 1 {
		t.Errorf("usage events: got %d, want 1 aggregate estimate", counts[core.EventUsageRecorded])
	}

	var usage *core.Event
	for i := range result.Events {
		if result.Events[i].Type == core.EventUsageRecorded {
			usage = &result.Events[i]
		}
		if result.Events[i].Subject.PersonID != "person:test" {
			t.Errorf("event %s subject not stamped", result.Events[i].ID)
		}
	}
	if usage == nil {
		t.Fatal("missing usage event")
	}
	if usage.Context.Workspace != "/repo" {
		t.Errorf("workspace: got %q", usage.Context.Workspace)
	}
	if got := usage.Attributes["token_estimation_method"]; got != "chars_per_token" {
		t.Errorf("estimation method: got %v", got)
	}
	if got := usage.Attributes[core.AttrInputTokens]; got == int64(0) {
		t.Errorf("input token estimate was zero")
	}
	if got := usage.Attributes[core.AttrOutputTokens]; got == int64(0) {
		t.Errorf("output token estimate was zero")
	}
}

func TestCollectArchiveWithMetadata(t *testing.T) {
	dir := t.TempDir()
	archiveDir := filepath.Join(dir, "sessions")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatal(err)
	}
	archive := `{"conversation_id":"meta01","cwd":"/repo","created_at":1782032400000,"updated_at":1782032460000,"value":{"conversation_id":"meta01","history":[{"user":{"content":"hello world"},"assistant":{"content":"done"},"request_metadata":{"request_start_timestamp_ms":1782032405000,"model_id":"claude-sonnet-4","time_between_chunks":[1,2],"tool_use_ids_and_names":[["toolu_1","Read"]]}}]}}`
	if err := os.WriteFile(filepath.Join(archiveDir, "meta01.json"), []byte(archive), 0o600); err != nil {
		t.Fatal(err)
	}

	c, err := New(Config{DBPath: filepath.Join(dir, "missing.sqlite3"), SessionsDir: archiveDir})
	if err != nil {
		t.Fatal(err)
	}
	result, err := c.Collect(context.Background(), core.CollectRequest{})
	if err != nil {
		t.Fatal(err)
	}

	counts := countByType(result.Events)
	if counts[core.EventUsageRecorded] != 1 {
		t.Errorf("usage events: got %d, want 1", counts[core.EventUsageRecorded])
	}
	if counts[core.EventToolCompleted] != 1 {
		t.Errorf("tool events: got %d, want 1", counts[core.EventToolCompleted])
	}
	for _, e := range result.Events {
		if e.Type == core.EventUsageRecorded {
			if got := e.Attributes[core.AttrModel]; got != "claude-sonnet-4" {
				t.Errorf("model: got %v", got)
			}
			if got := e.Attributes[core.AttrOutputTokens]; got != int64(2) {
				t.Errorf("output chunks: got %v, want 2", got)
			}
		}
	}
}

func TestMetadataFromTurn(t *testing.T) {
	raw := []byte(`{"user":{"content":"hello world"},"assistant":{"content":"done"},"request_metadata":{"request_start_timestamp_ms":1782032405000,"model_id":"claude-sonnet-4","time_between_chunks":[1,2],"tool_use_ids_and_names":[["toolu_1","Read"]]}}`)
	meta, ok := metadataFromTurn(raw)
	if !ok {
		t.Fatal("metadata not found")
	}
	if meta.ModelID != "claude-sonnet-4" {
		t.Fatalf("model: got %q", meta.ModelID)
	}
	if len(meta.TimeBetweenChunks) != 2 {
		t.Fatalf("chunks: got %d", len(meta.TimeBetweenChunks))
	}
}

func createKiroDB(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close() //nolint:errcheck
	if _, err := db.Exec(`CREATE TABLE conversations (key TEXT PRIMARY KEY, value TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE conversations_v2 (key TEXT NOT NULL, conversation_id TEXT NOT NULL, value TEXT NOT NULL, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL, PRIMARY KEY(key, conversation_id))`); err != nil {
		t.Fatal(err)
	}
	value := `{"conversation_id":"pair01","history":[[{"content":"please inspect this repository","images":[]},{"Response":{"content":"I inspected it."}}],[{"content":"now summarize"},{"ToolUse":{"name":"Read"}},{"Response":{"content":"Summary complete."}}]],"latest_summary":[]}`
	if _, err := db.Exec(`INSERT INTO conversations_v2 (key, conversation_id, value, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"/repo", "pair01", value, int64(1782032400000), int64(1782032460000)); err != nil {
		t.Fatal(err)
	}
}

func countByType(events []core.Event) map[core.EventType]int {
	counts := map[core.EventType]int{}
	for _, e := range events {
		counts[e.Type]++
	}
	return counts
}
