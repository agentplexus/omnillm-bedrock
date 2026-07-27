package omnidevx

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	core "github.com/plexusone/omnidevx-core"
)

type snapshot struct {
	ConversationID string          `json:"conversation_id"`
	CWD            string          `json:"cwd"`
	CreatedAt      int64           `json:"created_at"`
	UpdatedAt      int64           `json:"updated_at"`
	Value          json.RawMessage `json:"value"`
}

func (c *Collector) collectArchives(ctx context.Context, req core.CollectRequest, result *core.CollectionResult, seen map[string]int64) error {
	entries, err := os.ReadDir(c.sessionsDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, core.Diagnostic{
			Severity: core.SeverityWarning,
			Message:  fmt.Sprintf("read archive directory: %v", err),
			Path:     c.sessionsDir,
		})
		return nil
	}

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(c.sessionsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			result.Diagnostics = append(result.Diagnostics, core.Diagnostic{Severity: core.SeverityWarning, Message: fmt.Sprintf("read archive: %v", err), Path: path})
			continue
		}
		var snap snapshot
		if err := json.Unmarshal(data, &snap); err != nil {
			result.Diagnostics = append(result.Diagnostics, core.Diagnostic{Severity: core.SeverityWarning, Message: fmt.Sprintf("parse archive: %v", err), Path: path})
			continue
		}
		if snap.UpdatedAt == 0 {
			snap.UpdatedAt = fileModUnixMS(path)
		}
		if snap.CreatedAt == 0 {
			snap.CreatedAt = snap.UpdatedAt
		}
		if seen[snap.ConversationID] >= snap.UpdatedAt && snap.UpdatedAt > 0 {
			continue
		}
		events, diags := c.parseSnapshot(snap, req, path)
		result.Events = append(result.Events, events...)
		result.Diagnostics = append(result.Diagnostics, diags...)
		seen[snap.ConversationID] = snap.UpdatedAt
	}
	return nil
}
