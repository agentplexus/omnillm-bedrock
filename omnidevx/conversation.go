package omnidevx

import (
	"encoding/json"
	"fmt"
	"time"

	core "github.com/plexusone/omnidevx-core"
)

type conversation struct {
	ConversationID string            `json:"conversation_id"`
	History        []json.RawMessage `json:"history"`
	LatestSummary  json.RawMessage   `json:"latest_summary"`
}

type requestMetadata struct {
	RequestStartTimestampMS int64             `json:"request_start_timestamp_ms"`
	ModelID                 string            `json:"model_id"`
	TimeBetweenChunks       []json.RawMessage `json:"time_between_chunks"`
	ToolUseIDsAndNames      [][]string        `json:"tool_use_ids_and_names"`
}

type turnUsage struct {
	InputTokens  int64
	OutputTokens int64
	CacheRead    int64
	CacheWrite   int64
	Timestamp    time.Time
	Model        string
	ToolNames    []string
	HasMeta      bool
}

func (c *Collector) parseSnapshot(snap snapshot, req core.CollectRequest, path string) ([]core.Event, []core.Diagnostic) {
	if snap.ConversationID == "" {
		return nil, []core.Diagnostic{{Severity: core.SeverityWarning, Message: "conversation has no id", Path: path}}
	}
	var conv conversation
	if err := json.Unmarshal(snap.Value, &conv); err != nil {
		return nil, []core.Diagnostic{{Severity: core.SeverityWarning, Message: fmt.Sprintf("parse conversation value: %v", err), Path: path}}
	}
	if conv.ConversationID == "" {
		conv.ConversationID = snap.ConversationID
	}

	created, updated := snap.CreatedAt, snap.UpdatedAt
	fCreated, fUpdated := conversationBounds(conv, 0)
	if created == 0 {
		created = fCreated
	}
	if updated == 0 {
		updated = fUpdated
	}

	ctx := core.EventContext{SessionID: snap.ConversationID, Workspace: snap.CWD}
	var events []core.Event
	if ts := msTime(created); !ts.IsZero() && req.Period.Contains(ts) {
		events = append(events, core.Event{
			ID:         "kiro-cli:" + snap.ConversationID + ":start",
			Type:       core.EventSessionStarted,
			Timestamp:  ts,
			Source:     source,
			Context:    ctx,
			Provenance: provenance(),
		})
	}

	usages := c.estimateUsages(conv)
	var totalInput, totalOutput, totalCacheRead, totalCacheWrite int64
	for i, u := range usages {
		totalInput += u.InputTokens
		totalOutput += u.OutputTokens
		totalCacheRead += u.CacheRead
		totalCacheWrite += u.CacheWrite
		ts := u.Timestamp
		if ts.IsZero() {
			continue
		}
		if !req.Period.Contains(ts) {
			continue
		}
		events = append(events, usageEvent(snap.ConversationID, i, ts, ctx, u))
		for j, tool := range u.ToolNames {
			events = append(events, toolEvent(snap.ConversationID, i, j, ts, ctx, tool))
		}
	}
	if len(usages) > 0 && !hasTimedUsage(usages) {
		ts := msTime(updated)
		if !ts.IsZero() && req.Period.Contains(ts) {
			u := turnUsage{
				InputTokens:  totalInput,
				OutputTokens: totalOutput,
				CacheRead:    totalCacheRead,
				CacheWrite:   totalCacheWrite,
			}
			events = append(events, usageEvent(snap.ConversationID, -1, ts, ctx, u))
		}
	}

	if ts := msTime(updated); !ts.IsZero() && req.Period.Contains(ts) {
		events = append(events, core.Event{
			ID:        "kiro-cli:" + snap.ConversationID + ":end",
			Type:      core.EventSessionEnded,
			Timestamp: ts,
			Source:    source,
			Context:   ctx,
			Attributes: map[string]any{
				core.AttrSessionTotalTokens: totalInput + totalOutput + totalCacheRead + totalCacheWrite,
			},
			Provenance: provenance(),
		})
	}
	return events, nil
}

func conversationBounds(conv conversation, fallbackMS int64) (int64, int64) {
	var first, last int64
	for _, raw := range conv.History {
		meta, ok := metadataFromTurn(raw)
		if !ok || meta.RequestStartTimestampMS == 0 {
			continue
		}
		ts := meta.RequestStartTimestampMS
		if first == 0 || ts < first {
			first = ts
		}
		if ts > last {
			last = ts
		}
	}
	if first == 0 {
		first = fallbackMS
	}
	if last == 0 {
		last = first
	}
	return first, last
}

func (c *Collector) estimateUsages(conv conversation) []turnUsage {
	usages := make([]turnUsage, 0, len(conv.History))
	summaryTokens := c.tokensForRaw(conv.LatestSummary)
	var cumulative, prevAssistant int64 = summaryTokens, 0
	for i, raw := range conv.History {
		userTokens, assistantTokens, toolNames := c.turnTextAndTools(raw)
		meta, hasMeta := metadataFromTurn(raw)
		outTokens := assistantTokens
		if hasMeta && len(meta.TimeBetweenChunks) > 0 {
			outTokens = int64(len(meta.TimeBetweenChunks))
		}
		cacheRead := int64(0)
		if i > 0 {
			cacheRead = cumulative
		}
		cacheWrite := userTokens
		if i > 0 {
			cacheWrite += prevAssistant
		}
		u := turnUsage{
			InputTokens:  userTokens,
			OutputTokens: outTokens,
			CacheRead:    cacheRead,
			CacheWrite:   cacheWrite,
			Model:        meta.ModelID,
			ToolNames:    append(toolNames, metadataTools(meta)...),
			HasMeta:      hasMeta,
		}
		if meta.RequestStartTimestampMS > 0 {
			u.Timestamp = msTime(meta.RequestStartTimestampMS)
		}
		usages = append(usages, u)
		cumulative += userTokens + assistantTokens
		prevAssistant = assistantTokens
	}
	return usages
}

func usageEvent(sessionID string, idx int, ts time.Time, ctx core.EventContext, u turnUsage) core.Event {
	attrs := map[string]any{
		core.AttrInputTokens:         u.InputTokens,
		core.AttrOutputTokens:        u.OutputTokens,
		core.AttrCacheReadTokens:     u.CacheRead,
		core.AttrCacheCreationTokens: u.CacheWrite,
	}
	if u.Model != "" {
		attrs[core.AttrModel] = u.Model
	}
	if !u.HasMeta {
		attrs["token_estimation_method"] = "chars_per_token"
	}
	return core.Event{
		ID:         fmt.Sprintf("kiro-cli:%s:usage:%d", sessionID, idx),
		Type:       core.EventUsageRecorded,
		Timestamp:  ts,
		Source:     source,
		Context:    ctx,
		Attributes: attrs,
		Provenance: provenance(),
	}
}

func toolEvent(sessionID string, turn, idx int, ts time.Time, ctx core.EventContext, tool string) core.Event {
	return core.Event{
		ID:        fmt.Sprintf("kiro-cli:%s:tool:%d:%d", sessionID, turn, idx),
		Type:      core.EventToolCompleted,
		Timestamp: ts,
		Source:    source,
		Context:   ctx,
		Attributes: map[string]any{
			core.AttrTool: tool,
		},
		Provenance: provenance(),
	}
}

func hasTimedUsage(usages []turnUsage) bool {
	for _, u := range usages {
		if !u.Timestamp.IsZero() {
			return true
		}
	}
	return false
}
