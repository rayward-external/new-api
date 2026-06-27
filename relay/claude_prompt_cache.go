package relay

import (
	"os"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/gin-gonic/gin"
)

const (
	anthropicPromptCacheTTLEnv         = dto.AnthropicPromptCacheTTLEnv
	anthropicPromptCacheTTLHeader      = dto.AnthropicPromptCacheTTLHeader
	anthropicPromptCacheWorkloadHeader = dto.AnthropicPromptCacheWorkloadHeader
)

func applyAnthropicPromptCacheControlToClaudeRequest(c *gin.Context, req *dto.ClaudeRequest) bool {
	return dto.ApplyAnthropicPromptCacheControlToClaudeRequest(req, resolveAnthropicPromptCacheTTL(c))
}

func applyAnthropicPromptCacheControlToRawClaudeBody(c *gin.Context, body []byte) ([]byte, bool, error) {
	return dto.ApplyAnthropicPromptCacheControlToRawClaudeBody(body, resolveAnthropicPromptCacheTTL(c))
}

func applyAnthropicPromptCacheControlToBodyStorage(c *gin.Context, storage common.BodyStorage) (common.BodyStorage, bool, error) {
	body, err := storage.Bytes()
	if err != nil {
		return storage, false, err
	}
	out, changed, err := applyAnthropicPromptCacheControlToRawClaudeBody(c, body)
	if err != nil || !changed {
		return storage, changed, err
	}
	newStorage, err := common.CreateBodyStorage(out)
	if err != nil {
		return storage, false, err
	}
	_ = storage.Close()
	c.Set(common.KeyBodyStorage, newStorage)
	return newStorage, true, nil
}

func resolveAnthropicPromptCacheTTL(c *gin.Context) string {
	headerTTL := ""
	workload := ""
	if c != nil && c.Request != nil {
		headerTTL = c.GetHeader(anthropicPromptCacheTTLHeader)
		workload = c.GetHeader(anthropicPromptCacheWorkloadHeader)
	}
	if headerTTL == "" {
		headerTTL = os.Getenv(anthropicPromptCacheTTLEnv)
	}
	return dto.NormalizeAnthropicPromptCacheTTL(headerTTL, workload)
}
