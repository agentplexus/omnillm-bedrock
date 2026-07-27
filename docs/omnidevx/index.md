# OmniDevX: Kiro CLI Collector

The `omnidevx` package collects developer-experience telemetry from
**AWS Kiro CLI local stores** for the
[OmniDevX](https://github.com/plexusone/omnidevx-core) domain.

It lives in this repository rather than `omnidevx-core/providers` because
reading Kiro's live local database requires a SQLite driver
(`modernc.org/sqlite`). That keeps `omnidevx-core` dependency-light while
letting vendor-specific collectors live beside the rest of their vendor
adapters.

## What it reads

Kiro CLI stores conversation history locally:

- **SQLite database**:
  - macOS: `~/Library/Application Support/kiro-cli/data.sqlite3`
  - Linux: `~/.local/share/kiro-cli/data.sqlite3`
- **Optional archived snapshots**: `~/.kiro_sessions/*.json`, compatible
  with `kiro-usage`-style archivers.

The collector reads conversation IDs, workspace paths, session timestamps
when available, tool names, and estimated token usage. Prompt text, assistant
responses, and tool payloads are not copied into OmniDevX events.

## Format stability

Kiro's local formats are internal and have already appeared in multiple
shapes. The collector handles both:

- `conversations_v2`, with `created_at` and `updated_at` columns.
- `conversations`, where timestamps may be absent and the collector falls
  back to database/archive modification time.
- History items shaped as metadata-rich turn objects.
- History items shaped as user/assistant/tool object pairs.

When exact request metadata is unavailable, token usage is estimated from
text length using a configurable characters-per-token ratio. Estimated usage
events include `token_estimation_method=chars_per_token` and carry reduced
history-mode confidence.

## Usage

```go
import (
    kiro "github.com/plexusone/omni-aws/omnidevx"
    core "github.com/plexusone/omnidevx-core"
)

collector, err := kiro.New(kiro.Config{
    DBPath:      "", // default OS-specific Kiro CLI database
    SessionsDir: "", // default ~/.kiro_sessions
})
result, err := collector.Collect(ctx, core.CollectRequest{
    Period:  core.Period{Start: weekStart, End: weekEnd},
    Subject: core.SubjectRef{PersonID: "person:jane"},
})
```

## Events emitted

| Event | Source |
|-------|--------|
| `ai.session.started` / `ai.session.ended` | Conversation bounds from SQLite/archive metadata or fallback file mtime |
| `ai.usage.recorded` | Per-turn usage when request timestamps exist, otherwise aggregate estimated usage at session update time |
| `ai.tool.completed` | Tool names from request metadata or pair-shaped `ToolUse` records |

Kiro's exact prompt/completion/cache accounting is not always present in the
local records, so this provider should be treated as a historical estimator
unless future Kiro versions expose exact token fields locally.
