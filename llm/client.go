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

// LLMResponse LLM 返回的结构化响应
type LLMResponse struct {
	Cmd        string `json:"cmd"`         // 要执行的 Shell 命令
	Explain    string `json:"explain"`     // 命令的中文解释
	NeedSearch bool   `json:"need_search"` // 是否需要联网搜索
}

// systemPrompt 约束 LLM 严格按 JSON 格式返回的提示词（快路径）
const systemPrompt = `你是一个 Shell 命令翻译助手。用户会用自然语言描述操作需求，你需要将其翻译成精确的 Shell 命令。

你必须严格按以下 JSON 格式返回，不要包含 Markdown 代码块标记（不要用反引号包裹），仅输出纯 JSON：

{
  "cmd": "要执行的精确 shell 命令",
  "explain": "用中文简明扼要地解释该命令的含义和参数",
  "need_search": false
}

重要规则：
1. 如果用户的需求是常见操作（如文件操作、进程管理、网络查询等），请直接给出准确的命令，并将 need_search 设为 false。
2. 如果你不确定某个命令的正确写法，或者用户的请求涉及以下任何情况，请**务必**将 cmd 留空、need_search 设为 true：
   - 特定云平台 CLI（如 aws、gcloud、az、aliyun、tccli 等）
   - 非标准/生僻工具的特定参数或标志
   - 需要特定 API 端点、资源名称或版本相关的命令语法
   - 涉及你不完全确定正确性的复杂管道或多步骤操作
   即使你大体知道方向，只要对任何参数不完全确定，就应该触发搜索。诚实比给出错误命令更重要。
3. cmd 必须是在当前操作系统环境下可以直接执行的命令。
4. explain 必须使用中文。`

// searchSystemPrompt 用于慢路径（第二次 LLM 调用）：要求模型基于搜索结果生成命令
const searchSystemPrompt = `你是一个 Shell 命令翻译助手。用户会用自然语言描述操作需求，你需要将其翻译成精确的 Shell 命令。

你现在已经拥有来自网络搜索的相关参考资料。请仔细阅读搜索结果，提取正确的命令语法和参数，然后生成准确的 Shell 命令。

你必须严格按以下 JSON 格式返回，不要包含 Markdown 代码块标记（不要用反引号包裹），仅输出纯 JSON：

{
  "cmd": "基于搜索结果生成的精确 shell 命令",
  "explain": "用中文简明扼要地解释该命令的含义和参数",
  "need_search": false
}

重要规则：
1. 你必须基于搜索结果中的信息来构造命令，不要凭空猜测。
2. cmd 必须是在当前操作系统环境下可以直接执行的命令。
3. need_search 必须设置为 false，因为这是最终响应。
4. 如果搜索结果仍然不足以构造准确的命令，请在 cmd 中给出最佳尝试，并在 explain 中说明不确定之处。
5. explain 必须使用中文。`

// CallLLM 调用 LLM API，将用户的自然语言请求翻译为 Shell 命令（快路径）
// apiBaseURL: API 端点地址（如 https://api.openai.com/v1 或 https://api.deepseek.com）
// apiKey: API 密钥
// modelName: 模型名称（如 gpt-4o-mini、deepseek-v4-flash）
// context: GatherContext() 收集的系统环境信息
// userQuery: 用户的自然语言请求
func CallLLM(apiBaseURL string, apiKey string, modelName string, context string, userQuery string) (*LLMResponse, error) {
	userMsg := fmt.Sprintf("以下是当前的系统环境信息：\n\n%s\n\n用户的自然语言请求：%s", context, userQuery)
	return callLLMInternal(apiBaseURL, apiKey, modelName, systemPrompt, userMsg)
}

// CallLLMWithSearch 调用 LLM API 并传入网络搜索结果，让模型基于搜索结果生成命令（慢路径）
// searchResults: FlattenResults() 产出的搜索结果文本摘要
func CallLLMWithSearch(apiBaseURL string, apiKey string, modelName string, context string, userQuery string, searchResults string) (*LLMResponse, error) {
	userMsg := fmt.Sprintf(
		"以下是当前的系统环境信息：\n\n%s\n\n用户的自然语言请求：%s\n\n"+
			"以下是从网络上搜索到的相关参考资料，请根据这些结果构造准确的命令：\n\n%s",
		context, userQuery, searchResults,
	)
	return callLLMInternal(apiBaseURL, apiKey, modelName, searchSystemPrompt, userMsg)
}

// callLLMInternal LLM 调用的内部实现，发送请求并解析 JSON 响应
func callLLMInternal(apiBaseURL string, apiKey string, modelName string, systemPrompt string, userMessage string) (*LLMResponse, error) {
	// 拼接完整的 Chat Completions 端点
	apiURL := strings.TrimRight(apiBaseURL, "/") + "/chat/completions"

	reqBody := chatCompletionRequest{
		Model: modelName,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMessage},
		},
		Temperature: 0.1, // 低温度以保证输出稳定
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	// 设置 60 秒超时，等待 LLM 生成响应
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API 请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回错误状态 %d: %s", resp.StatusCode, string(body))
	}

	var chatResp chatCompletionResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("解析 API 响应失败: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("API 返回的 choices 为空")
	}

	// 提取 LLM 返回的 JSON 内容并解析
	content := chatResp.Choices[0].Message.Content
	content = strings.TrimSpace(content)

	var llmResp LLMResponse
	if err := json.Unmarshal([]byte(content), &llmResp); err != nil {
		return nil, fmt.Errorf("解析 LLM 返回的 JSON 失败: %w\n原始内容: %s", err, content)
	}

	return &llmResp, nil
}

// chatCompletionRequest OpenAI 兼容的 Chat Completions 请求体
type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

// chatMessage 单条对话消息
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatCompletionResponse OpenAI 兼容的 Chat Completions 响应体
type chatCompletionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}
