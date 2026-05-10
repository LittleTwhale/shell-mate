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
	Explain    string `json:"explain"`     // 命令的解释说明
	NeedSearch bool   `json:"need_search"` // 是否需要联网搜索
}

// ========== 多语言系统提示词 ==========

// prompts 按语言存储系统提示词和用户消息模板
type promptSet struct {
	systemPrompt       string // 快路径系统提示词
	searchSystemPrompt string // 慢路径系统提示词（含搜索结果）
	userMsgTmpl        string // 用户消息模板（含系统上下文）
	userMsgSearchTmpl  string // 用户消息模板（含系统上下文 + 搜索结果）
}

var prompts = map[string]promptSet{
	"zh": {
		systemPrompt: `你是一个 Shell 命令翻译助手。用户会用自然语言描述操作需求，你需要将其翻译成精确的 Shell 命令。

你必须严格按以下 JSON 格式返回，不要包含 Markdown 代码块标记（不要用反引号包裹），仅输出纯 JSON：

{
  "cmd": "要执行的精确 shell 命令",
  "explain": "用中文简明扼要地解释该命令的含义和参数",
  "need_search": false
}

重要规则：
1. 如果用户的需求是常见操作（如文件操作、进程管理、网络查询等），请直接给出准确的命令，并将 need_search 设为 false。
2. 如果你不确定某个命令的正确写法，或者用户的请求涉及以下任何情况，请务必将 cmd 留空、need_search 设为 true：
   - 特定云平台 CLI（如 aws、gcloud、az、aliyun、tccli 等）
   - 非标准/生僻工具的特定参数或标志
   - 需要特定 API 端点、资源名称或版本相关的命令语法
   - 涉及你不完全确定正确性的复杂管道或多步骤操作
   即使你大体知道方向，只要对任何参数不完全确定，就应该触发搜索。诚实比给出错误命令更重要。
3. cmd 必须是在当前操作系统环境下可以直接执行的命令。
4. explain 必须使用中文。`,

		searchSystemPrompt: `你是一个 Shell 命令翻译助手。用户会用自然语言描述操作需求，你需要将其翻译成精确的 Shell 命令。

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
5. explain 必须使用中文。`,

		userMsgTmpl:       "以下是当前的系统环境信息：\n\n%s\n\n用户的自然语言请求：%s",
		userMsgSearchTmpl: "以下是当前的系统环境信息：\n\n%s\n\n用户的自然语言请求：%s\n\n以下是从网络上搜索到的相关参考资料，请根据这些结果构造准确的命令：\n\n%s",
	},
	"en": {
		systemPrompt: `You are a Shell command translator. The user describes what they want to do in natural language, and you translate it into a precise Shell command.

You MUST return strictly valid JSON in the following format, without Markdown code fences (no backticks), output ONLY raw JSON:

{
  "cmd": "the exact shell command to execute",
  "explain": "a concise explanation of what the command does and its arguments",
  "need_search": false
}

Important rules:
1. For common operations (file manipulation, process management, network queries, etc.), provide the exact command and set need_search to false.
2. If you are unsure about the correct syntax, or the request involves any of the following, you MUST leave cmd empty and set need_search to true:
   - Cloud platform CLIs (aws, gcloud, az, aliyun, tccli, etc.)
   - Non-standard or obscure tools with specific flags or arguments
   - Commands requiring specific API endpoints, resource names, or version-dependent syntax
   - Complex pipelines or multi-step operations whose correctness you cannot fully verify
   Being honest and triggering a search is better than giving a wrong command.
3. The cmd must be directly executable in the current OS environment.
4. explain must be in English.`,

		searchSystemPrompt: `You are a Shell command translator. The user describes what they want to do in natural language, and you translate it into a precise Shell command.

You now have access to reference material from a web search. Read the search results carefully, extract the correct command syntax and arguments, then generate an accurate Shell command.

You MUST return strictly valid JSON in the following format, without Markdown code fences (no backticks), output ONLY raw JSON:

{
  "cmd": "the exact shell command based on search results",
  "explain": "a concise explanation of what the command does and its arguments",
  "need_search": false
}

Important rules:
1. Base the command on information from the search results — do not guess.
2. The cmd must be directly executable in the current OS environment.
3. need_search must be set to false, as this is the final response.
4. If the search results are still insufficient for an accurate command, give your best attempt in cmd and note the uncertainties in explain.
5. explain must be in English.`,

		userMsgTmpl:       "Current system environment:\n\n%s\n\nUser's natural language request: %s",
		userMsgSearchTmpl: "Current system environment:\n\n%s\n\nUser's natural language request: %s\n\nThe following reference material was found via web search; use it to construct the correct command:\n\n%s",
	},
}

// getPromptSet 根据语言获取提示词集合，默认为中文
func getPromptSet(lang string) promptSet {
	if ps, ok := prompts[lang]; ok {
		return ps
	}
	return prompts["zh"]
}

// CallLLM 调用 LLM API，将用户的自然语言请求翻译为 Shell 命令（快路径）
// apiBaseURL: API 端点地址（如 https://api.openai.com/v1 或 https://api.deepseek.com）
// apiKey: API 密钥
// modelName: 模型名称（如 gpt-4o-mini、deepseek-v4-flash）
// context: GatherContext() 收集的系统环境信息
// userQuery: 用户的自然语言请求
// lang: 当前语言 (zh/en)
func CallLLM(apiBaseURL string, apiKey string, modelName string, context string, userQuery string, lang string) (*LLMResponse, error) {
	ps := getPromptSet(lang)
	userMsg := fmt.Sprintf(ps.userMsgTmpl, context, userQuery)
	return callLLMInternal(apiBaseURL, apiKey, modelName, ps.systemPrompt, userMsg)
}

// CallLLMWithSearch 调用 LLM API 并传入网络搜索结果，让模型基于搜索结果生成命令（慢路径）
// searchResults: FlattenResults() 产出的搜索结果文本摘要
func CallLLMWithSearch(apiBaseURL string, apiKey string, modelName string, context string, userQuery string, searchResults string, lang string) (*LLMResponse, error) {
	ps := getPromptSet(lang)
	userMsg := fmt.Sprintf(ps.userMsgSearchTmpl, context, userQuery, searchResults)
	return callLLMInternal(apiBaseURL, apiKey, modelName, ps.searchSystemPrompt, userMsg)
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
		return nil, fmt.Errorf("serialize request failed: %w", err)
	}

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create HTTP request failed: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	// 设置 60 秒超时，等待 LLM 生成响应
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned error status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp chatCompletionResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("parse API response failed: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("API returned empty choices")
	}

	// 提取 LLM 返回的 JSON 内容并解析
	content := chatResp.Choices[0].Message.Content
	content = strings.TrimSpace(content)

	var llmResp LLMResponse
	if err := json.Unmarshal([]byte(content), &llmResp); err != nil {
		return nil, fmt.Errorf("parse LLM JSON response failed: %w\nraw content: %s", err, content)
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
