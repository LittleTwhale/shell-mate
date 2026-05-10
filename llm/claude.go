package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ClaudeProvider 实现 Anthropic Messages API
type ClaudeProvider struct {
	apiBaseURL string
	apiKey     string
	modelName  string
}

// Name 返回 Provider 名称
func (p *ClaudeProvider) Name() string {
	return "claude"
}

// Chat 调用 Anthropic Messages API 并解析响应
func (p *ClaudeProvider) Chat(systemPrompt, userMessage string) (*LLMResponse, error) {
	apiURL := strings.TrimRight(p.apiBaseURL, "/") + "/messages"

	reqBody := claudeRequest{
		Model:      p.modelName,
		MaxTokens:  1024,
		System:     systemPrompt,
		Messages:   []claudeMessage{{Role: "user", Content: userMessage}},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化 Claude 请求失败: %w", err)
	}

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("创建 Claude HTTP 请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Claude API 请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 Claude 响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Claude API 返回错误状态 %d: %s", resp.StatusCode, string(body))
	}

	var claudeResp claudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return nil, fmt.Errorf("解析 Claude API 响应失败: %w", err)
	}

	if len(claudeResp.Content) == 0 {
		return nil, fmt.Errorf("Claude API 返回了空的 content")
	}

	// 提取第一个 text 块的内容，去除可能的 Markdown 代码块包裹
	content := strings.TrimSpace(claudeResp.Content[0].Text)
	content = stripMarkdownCodeBlock(content)

	var llmResp LLMResponse
	if err := json.Unmarshal([]byte(content), &llmResp); err != nil {
		return nil, fmt.Errorf("解析 Claude JSON 响应失败: %w\n原始内容: %s", err, content)
	}

	return &llmResp, nil
}

// stripMarkdownCodeBlock 去除可能的 Markdown 代码块标记（```json ... ```）
func stripMarkdownCodeBlock(s string) string {
	// 去掉开头的 ```json 或 ```
	if strings.HasPrefix(s, "```") {
		// 跳过第一行（```json 或 ```）
		newline := strings.Index(s, "\n")
		if newline != -1 {
			s = s[newline+1:]
		}
	}
	// 去掉末尾的 ```
	if strings.HasSuffix(s, "```") {
		s = s[:len(s)-3]
	}
	// 去掉可能紧随 ``` 的换行符
	s = strings.TrimRight(s, "\n\r ")
	return s
}

// ========== Claude API 数据结构 ==========

// claudeRequest Anthropic Messages API 请求体
type claudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	System    string          `json:"system"`
	Messages  []claudeMessage `json:"messages"`
}

// claudeMessage 单条消息
type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// claudeResponse Anthropic Messages API 响应体
type claudeResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
}
