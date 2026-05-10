package llm

import "fmt"

// Provider LLM 厂商适配器接口，每种厂商实现自己的 Chat 方法
type Provider interface {
	// Name 返回 Provider 名称标识（如 openai / deepseek / ollama / claude）
	Name() string
	// Chat 发送系统提示词和用户消息，返回结构化 LLM 响应
	Chat(systemPrompt, userMessage string) (*LLMResponse, error)
}

// Preset 每个 Provider 的默认配置预设
type Preset struct {
	DefaultBaseURL string // 默认 API 端点
	DefaultModel   string // 默认模型名称
}

// Presets 所有内置 Provider 预设表
var Presets = map[string]Preset{
	"openai": {
		DefaultBaseURL: "https://api.openai.com/v1",
		DefaultModel:   "gpt-4o-mini",
	},
	"deepseek": {
		DefaultBaseURL: "https://api.deepseek.com",
		DefaultModel:   "deepseek-v4-flash",
	},
	"ollama": {
		DefaultBaseURL: "http://localhost:11434/v1",
		DefaultModel:   "llama3.2",
	},
	"claude": {
		DefaultBaseURL: "https://api.anthropic.com/v1",
		DefaultModel:   "claude-sonnet-4-6",
	},
}

// NewProvider 根据 provider 名称创建对应的 Provider 实例
// providerName: openai / deepseek / ollama / claude
// apiBaseURL / apiKey / modelName: 用户配置或预设的默认值
func NewProvider(providerName, apiBaseURL, apiKey, modelName string) (Provider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key 不能为空")
	}

	switch providerName {
	case "claude":
		return &ClaudeProvider{
			apiBaseURL: apiBaseURL,
			apiKey:     apiKey,
			modelName:  modelName,
		}, nil
	default:
		// openai / deepseek / ollama 均使用 OpenAI 兼容格式
		return &OpenAICompatibleProvider{
			apiBaseURL: apiBaseURL,
			apiKey:     apiKey,
			modelName:  modelName,
			name:       providerName,
		}, nil
	}
}
