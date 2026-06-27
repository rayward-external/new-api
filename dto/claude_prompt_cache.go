package dto

import (
	"encoding/json"
	"strings"

	"github.com/tidwall/sjson"
)

const (
	AnthropicPromptCacheTTLEnv         = "ANTHROPIC_PROMPT_CACHE_TTL"
	AnthropicPromptCacheTTLHeader      = "x-anthropic-prompt-cache-ttl"
	AnthropicPromptCacheWorkloadHeader = "x-anthropic-prompt-cache-workload"
	anthropicPromptCacheControlType    = "ephemeral"
)

func ApplyAnthropicPromptCacheControlToClaudeRequest(req *ClaudeRequest, ttl string) bool {
	if req == nil || ttl == "" || ClaudeRequestHasCacheControl(req) {
		return false
	}
	req.CacheControl = newAnthropicPromptCacheControlRaw(ttl)
	return true
}

func ApplyAnthropicPromptCacheControlToRawClaudeBody(body []byte, ttl string) ([]byte, bool, error) {
	if ttl == "" || RawClaudeBodyHasCacheControl(body) {
		return body, false, nil
	}
	cc := newAnthropicPromptCacheControlRaw(ttl)
	out, err := sjson.SetRawBytes(body, "cache_control", cc)
	if err != nil {
		return body, false, err
	}
	return out, true, nil
}

func NormalizeAnthropicPromptCacheTTL(value, workload string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "off", "false", "none", "disabled":
		return ""
	case "5m":
		return "5m"
	case "1h":
		return "1h"
	case "auto":
		if IsLongRunningAnthropicWorkload(workload) {
			return "1h"
		}
		return "5m"
	default:
		return ""
	}
}

func IsLongRunningAnthropicWorkload(workload string) bool {
	switch strings.ToLower(strings.TrimSpace(workload)) {
	case "eval", "evaluation", "benchmark", "bench", "batch", "pipeline", "long", "long-running":
		return true
	default:
		return false
	}
}

func ClaudeRequestHasCacheControl(req *ClaudeRequest) bool {
	if req == nil {
		return false
	}
	if len(req.CacheControl) > 0 {
		return true
	}
	data, err := json.Marshal(req)
	if err != nil {
		return false
	}
	return RawClaudeBodyHasCacheControl(data)
}

func RawClaudeBodyHasCacheControl(body []byte) bool {
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return false
	}
	return hasCacheControlKey(value)
}

func hasCacheControlKey(value any) bool {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			if key == "cache_control" && child != nil {
				return true
			}
			if hasCacheControlKey(child) {
				return true
			}
		}
	case []any:
		for _, child := range v {
			if hasCacheControlKey(child) {
				return true
			}
		}
	}
	return false
}

func newAnthropicPromptCacheControlRaw(ttl string) json.RawMessage {
	return json.RawMessage(`{"type":"` + anthropicPromptCacheControlType + `","ttl":"` + ttl + `"}`)
}
