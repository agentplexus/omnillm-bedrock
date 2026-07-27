package omnidevx

import (
	"encoding/base64"
	"encoding/json"
)

func (c *Collector) turnTextAndTools(raw json.RawMessage) (userTokens, assistantTokens int64, tools []string) {
	var parts []json.RawMessage
	if err := json.Unmarshal(raw, &parts); err == nil {
		for _, part := range parts {
			u, a, t := c.objectTextAndTools(part)
			userTokens += u
			assistantTokens += a
			tools = append(tools, t...)
		}
		return userTokens, assistantTokens, tools
	}
	return c.objectTextAndTools(raw)
}

func (c *Collector) objectTextAndTools(raw json.RawMessage) (userTokens, assistantTokens int64, tools []string) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return 0, c.tokensForRaw(raw), nil
	}
	for key, val := range obj {
		switch key {
		case "user", "content", "additional_context", "env_context":
			userTokens += c.tokensForRaw(val)
		case "assistant":
			assistantTokens += c.tokensForRaw(val)
		case "Response":
			assistantTokens += c.responseTokens(val)
		case "ToolUse":
			if name := toolName(val); name != "" {
				tools = append(tools, name)
			}
		case "images":
			userTokens += c.imageTokens(val)
		}
	}
	return userTokens, assistantTokens, tools
}

func (c *Collector) responseTokens(raw json.RawMessage) int64 {
	var obj struct {
		Content json.RawMessage `json:"content"`
	}
	if json.Unmarshal(raw, &obj) == nil && len(obj.Content) > 0 {
		return c.tokensForRaw(obj.Content)
	}
	return c.tokensForRaw(raw)
}

func (c *Collector) tokensForRaw(raw json.RawMessage) int64 {
	if len(raw) == 0 || string(raw) == "null" {
		return 0
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return int64(len(s)) / c.charsPerToken
	}
	return int64(len(raw)) / c.charsPerToken
}

func (c *Collector) imageTokens(raw json.RawMessage) int64 {
	var images []json.RawMessage
	if err := json.Unmarshal(raw, &images); err != nil {
		return 0
	}
	var total int64
	for _, image := range images {
		var obj map[string]any
		if json.Unmarshal(image, &obj) != nil {
			continue
		}
		total += 1600
		if src, ok := obj["source"].(map[string]any); ok {
			if b64, ok := src["Bytes"].(string); ok {
				if data, err := base64.StdEncoding.DecodeString(b64); err == nil {
					total += int64(len(data)) / 1500
				}
			}
		}
	}
	return total
}

func toolName(raw json.RawMessage) string {
	var obj map[string]any
	if json.Unmarshal(raw, &obj) != nil {
		return ""
	}
	for _, key := range []string{"name", "tool_name", "type"} {
		if s, ok := obj[key].(string); ok {
			return s
		}
	}
	return ""
}

func metadataFromTurn(raw json.RawMessage) (requestMetadata, bool) {
	if meta, ok := metadataFromObject(raw); ok {
		return meta, true
	}
	var parts []json.RawMessage
	if err := json.Unmarshal(raw, &parts); err != nil {
		return requestMetadata{}, false
	}
	for _, part := range parts {
		if meta, ok := metadataFromObject(part); ok {
			return meta, true
		}
	}
	return requestMetadata{}, false
}

func metadataFromObject(raw json.RawMessage) (requestMetadata, bool) {
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil {
		return requestMetadata{}, false
	}
	metaRaw, ok := obj["request_metadata"]
	if !ok || len(metaRaw) == 0 || string(metaRaw) == "null" {
		return requestMetadata{}, false
	}
	var meta requestMetadata
	if json.Unmarshal(metaRaw, &meta) != nil {
		return requestMetadata{}, false
	}
	return meta, true
}

func metadataTools(meta requestMetadata) []string {
	var tools []string
	for _, pair := range meta.ToolUseIDsAndNames {
		if len(pair) > 1 && pair[1] != "" {
			tools = append(tools, pair[1])
		}
	}
	return tools
}
