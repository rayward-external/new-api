package relay

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/gin-gonic/gin"
)

func TestApplyAnthropicPromptCacheControlToClaudeRequestInjectsOneHour(t *testing.T) {
	t.Setenv(anthropicPromptCacheTTLEnv, "1h")
	c := testClaudePromptCacheContext(nil)
	req := &dto.ClaudeRequest{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: uintPtr(1024),
		Messages:  []dto.ClaudeMessage{{Role: "user", Content: "hello"}},
	}

	changed := applyAnthropicPromptCacheControlToClaudeRequest(c, req)

	if !changed {
		t.Fatal("expected cache_control to be injected")
	}
	var got map[string]string
	if err := json.Unmarshal(req.CacheControl, &got); err != nil {
		t.Fatalf("unmarshal cache_control: %v", err)
	}
	if got["type"] != "ephemeral" || got["ttl"] != "1h" {
		t.Fatalf("cache_control = %#v, want ephemeral 1h", got)
	}
}

func TestApplyAnthropicPromptCacheControlToClaudeRequestPreservesClientControl(t *testing.T) {
	t.Setenv(anthropicPromptCacheTTLEnv, "1h")
	c := testClaudePromptCacheContext(nil)
	req := &dto.ClaudeRequest{
		Model:        "claude-sonnet-4-20250514",
		MaxTokens:    uintPtr(1024),
		CacheControl: json.RawMessage(`{"type":"ephemeral","ttl":"5m"}`),
		Messages:     []dto.ClaudeMessage{{Role: "user", Content: "hello"}},
	}

	changed := applyAnthropicPromptCacheControlToClaudeRequest(c, req)

	if changed {
		t.Fatal("expected existing cache_control to be preserved")
	}
	if string(req.CacheControl) != `{"type":"ephemeral","ttl":"5m"}` {
		t.Fatalf("cache_control = %s, want client value", string(req.CacheControl))
	}
}

func TestApplyAnthropicPromptCacheControlToClaudeRequestHeaderDisablesEnvDefault(t *testing.T) {
	t.Setenv(anthropicPromptCacheTTLEnv, "1h")
	c := testClaudePromptCacheContext(map[string]string{anthropicPromptCacheTTLHeader: "off"})
	req := &dto.ClaudeRequest{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: uintPtr(1024),
		Messages:  []dto.ClaudeMessage{{Role: "user", Content: "hello"}},
	}

	changed := applyAnthropicPromptCacheControlToClaudeRequest(c, req)

	if changed {
		t.Fatal("expected header off override to disable injection")
	}
	if len(req.CacheControl) != 0 {
		t.Fatalf("cache_control = %s, want empty", string(req.CacheControl))
	}
}

func TestApplyAnthropicPromptCacheControlToClaudeRequestAutoUsesWorkloadHeader(t *testing.T) {
	t.Setenv(anthropicPromptCacheTTLEnv, "auto")
	c := testClaudePromptCacheContext(map[string]string{anthropicPromptCacheWorkloadHeader: "benchmark"})
	req := &dto.ClaudeRequest{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: uintPtr(1024),
		Messages:  []dto.ClaudeMessage{{Role: "user", Content: "hello"}},
	}

	changed := applyAnthropicPromptCacheControlToClaudeRequest(c, req)

	if !changed {
		t.Fatal("expected cache_control to be injected")
	}
	var got map[string]string
	if err := json.Unmarshal(req.CacheControl, &got); err != nil {
		t.Fatalf("unmarshal cache_control: %v", err)
	}
	if got["ttl"] != "1h" {
		t.Fatalf("cache_control = %#v, want 1h for benchmark workload", got)
	}
}

func TestApplyAnthropicPromptCacheControlToRawClaudeBody(t *testing.T) {
	t.Setenv(anthropicPromptCacheTTLEnv, "1h")
	c := testClaudePromptCacheContext(nil)
	body := []byte(`{"model":"claude-sonnet-4-20250514","max_tokens":1024,"messages":[{"role":"user","content":"hello"}]}`)

	out, changed, err := applyAnthropicPromptCacheControlToRawClaudeBody(c, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected raw body to be changed")
	}
	var got struct {
		CacheControl map[string]string `json:"cache_control"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal raw body: %v", err)
	}
	if got.CacheControl["type"] != "ephemeral" || got.CacheControl["ttl"] != "1h" {
		t.Fatalf("cache_control = %#v, want ephemeral 1h", got.CacheControl)
	}
}

func TestApplyAnthropicPromptCacheControlToRawClaudeBodyPreservesNestedClientControl(t *testing.T) {
	t.Setenv(anthropicPromptCacheTTLEnv, "1h")
	c := testClaudePromptCacheContext(nil)
	body := []byte(`{"model":"claude-sonnet-4-20250514","max_tokens":1024,"messages":[{"role":"user","content":[{"type":"text","text":"hello","cache_control":{"type":"ephemeral"}}]}]}`)

	out, changed, err := applyAnthropicPromptCacheControlToRawClaudeBody(c, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed {
		t.Fatal("expected nested client cache_control to prevent gateway injection")
	}
	if string(out) != string(body) {
		t.Fatalf("body changed unexpectedly: %s", string(out))
	}
}

func testClaudePromptCacheContext(headers map[string]string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c := &gin.Context{}
	req, _ := http.NewRequest(http.MethodPost, "/v1/messages", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	c.Request = req
	return c
}

func uintPtr(v uint) *uint {
	return &v
}
