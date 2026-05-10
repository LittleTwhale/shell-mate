package cmd

import (
	"github.com/spf13/viper"
)

// Lang 语言类型
type Lang string

const (
	LangZH Lang = "zh" // 中文
	LangEN Lang = "en" // 英文
)

// messages 多语言翻译映射表
var messages = map[Lang]map[string]string{
	LangZH: {
		// ========== Cobra 命令元数据 ==========
		"root.use":   "sm [自然语言描述]",
		"root.short": "shell-mate — 终端命令行 AI 助手",
		"root.long": `shell-mate 是一个基于 Go 开发的终端命令行 AI 助手，
能将你的自然语言请求即时翻译成可执行的 Shell 命令。

━━━ 核心特性 ━━━
  ◆  环境感知     自动收集操作系统、Shell 类型、当前目录文件列表等上下文
  ◆  交互式 TUI    生成命令后提供美观的终端菜单，支持 [执行/取消/解释]
  ◆  危险命令拦截  检测 rm -rf、mkfs、dd 等危险操作，强制输入 YES 二次确认
  ◆  智能搜索      遇到云平台 CLI、生僻工具等不确定请求时，自动联网搜索防幻觉
  ◆  多语言支持    通过 sm config -l 自由切换中英文界面

━━━ 使用示例 ━━━
  sm 列出当前目录下最大的 5 个文件
  sm 查找并杀死占用 8080 端口的进程
  sm 用 AWS CLI 创建一个 S3 存储桶       （触发智能搜索）
  sm 递归删除所有 .log 文件               （触发危险拦截）
  sm 查看 git 最近 10 条提交的统计信息

━━━ 配置指南 ━━━
  sm config -k <API_KEY>            设置 LLM API 密钥（必填）
  sm config -b <BASE_URL>           设置 API 端点地址（默认 https://api.deepseek.com）
  sm config -m <MODEL>              设置模型名称（默认 deepseek-v4-flash）
  sm config -l <zh|en>              切换界面语言
  sm config --add-danger "word"     添加自定义高危拦截词
  sm config --remove-danger "word"  移除自定义高危拦截词
  sm config                         查看当前配置

━━━ 支持的环境变量 ━━━
  SHELL_MATE_API_KEY        等同于 sm config -k`,

		"config.short": "管理 shell-mate 配置项",
		"config.long": `设置 API_KEY、API_BASE_URL、MODEL_NAME、SEARCH_API_KEY、LANGUAGE 等配置项，
并将它们持久化保存到 ~/.shell-mate.yaml 文件中。

不带任何参数运行时，显示当前配置。`,

		// ========== TUI 交互菜单 ==========
		"tui.title":          "是否执行此命令?",
		"tui.description":    "命令: %s",
		"tui.opt_execute":    "[y] 执行 (Execute)",
		"tui.opt_cancel":     "[n] 取消 (Cancel)",
		"tui.opt_explain":    "[e] 解释 (Explain)",
		"tui.cancelled":      "已取消。",
		"tui.explain_prefix": "\n命令解释: %s\n\n",
		"tui.run_error":      "TUI 运行失败: %v",

		// 安全护栏 — 警告框
		"guard.warning_title":  "严重安全警告：检测到高危命令操作！",
		"guard.warning_desc":   "该命令可能对您的系统造成不可逆的损害！",
		"guard.warning_prompt": "请仔细确认无误后，输入 YES 以继续。",
		"guard.warning_line1":  "╔══════════════════════════════════════════════════════════════╗",
		"guard.warning_line2":  "║  ⚠  %-52s ║",
		"guard.warning_line3":  "╠══════════════════════════════════════════════════════════════╣",
		"guard.warning_line4":  "║  命令: %-48s ║",
		"guard.warning_line5":  "║  %-56s ║",
		"guard.warning_line6":  "╚══════════════════════════════════════════════════════════════╝",

		// 安全护栏 — YES 确认输入框
		"guard.confirm_title":       "请输入大写的 YES 以确认执行此高危命令",
		"guard.confirm_prompt":      "> ",
		"guard.confirm_cancel_hint": "按 Esc 取消",
		"guard.confirm_validate":    "请输入 YES（大写）以确认",
		"guard.confirm_cancelled":   "已取消高危命令执行。",

		// 配置显示
		"config.title":      "当前配置 (~/.shell-mate.yaml):",
		"config.api_key":    "  API_KEY        : %s",
		"config.api_base":   "  API_BASE_URL   : %s",
		"config.model_name": "  MODEL_NAME     : %s",
		"config.search_key": "  SEARCH_API_KEY : %s",
		"config.language":   "  LANGUAGE       : %s",
		"config.danger_list": "  DANGER_KEYWORDS: %v",
		"config.unset":      "(未设置)",
		"config.saved":      "配置已保存到 ~/.shell-mate.yaml",
		"config.write_err":  "写入配置文件失败: %s",
		"config.read_err":   "读取配置文件出错: %s",

		// root.go
		"root.config_missing": "错误: 请先设置 API_KEY，运行: sm config -k <your-api-key>",
		"root.llm_fail":       "调用 AI 失败: %v",
		"root.home_err":       "无法获取用户主目录:",
		"root.llm_nocmd":      "AI 无法生成有效命令，请尝试用不同方式描述您的需求。",

		// 命令执行
		"exec.run_error": "命令执行失败: %v",

		// 请求中 spinner 提示（用于旋转动画）
		"root.llm_calling":     "正在请求 AI 翻译...",
		"root.search_spin":     "遇到复杂指令，正在联网搜索...",
		"root.search_spin_done": "搜索完成，正在重新生成命令...",

		// Agentic 搜索（阶段 5 慢路径）
		"root.search_start":    "\n[i] 遇到复杂指令，正在联网搜索解决方案...",
		"root.search_done":     "[✓] 搜索完成，正在基于搜索结果重新生成命令...\n",
		"root.search_fail":     "[✗] 联网搜索失败: %v\n将使用 LLM 的直接回答作为备选。",
		"root.search_fallback": "[i] 搜索无结果，使用 LLM 直接回答。",
		"root.search_still":    "[!] 二次调用后 LLM 仍无法给出确切命令，以下是当前最佳结果：",
	},
	LangEN: {
		// ========== Cobra command metadata ==========
		"root.use":   "sm [natural language description]",
		"root.short": "shell-mate — CLI AI Assistant",
		"root.long": `shell-mate is a Go-based CLI AI assistant that instantly
translates natural language into executable Shell commands.

━━━ Core Features ━━━
  ◆  Context-Aware      Automatically collects OS, Shell type, and directory listing
  ◆  Interactive TUI     Beautiful terminal menu with execute / cancel / explain options
  ◆  Safety Guardrails   Detects dangerous commands (rm -rf, mkfs, dd, etc.) with YES confirmation
  ◆  Agentic Search      Auto-searches the web for complex/uncertain requests to prevent hallucination
  ◆  Multi-Language      Switch between Chinese and English UI via sm config -l

━━━ Examples ━━━
  sm list the 5 largest files in current directory
  sm find and kill the process using port 8080
  sm create an S3 bucket with AWS CLI          (triggers agentic search)
  sm recursively delete all .log files          (triggers safety guard)
  sm show git commit stats for the last 10 commits

━━━ Configuration ━━━
  sm config -k <API_KEY>                Set LLM API key (required)
  sm config -b <BASE_URL>               Set API endpoint (default https://api.deepseek.com)
  sm config -m <MODEL>                  Set model name (default deepseek-v4-flash)
  sm config -l <zh|en>                  Switch UI language
  sm config --add-danger "word"         Add a custom dangerous keyword
  sm config --remove-danger "word"      Remove a custom dangerous keyword
  sm config                             Show current configuration

━━━ Environment Variables ━━━
  SHELL_MATE_API_KEY        Equivalent to sm config -k`,

		"config.short": "Manage shell-mate configuration",
		"config.long": `Set API_KEY, API_BASE_URL, MODEL_NAME, SEARCH_API_KEY, LANGUAGE
and persist them to ~/.shell-mate.yaml.

Running without arguments displays current configuration.`,

		// ========== TUI interactive menu ==========
		"tui.title":          "Execute this command?",
		"tui.description":    "Command: %s",
		"tui.opt_execute":    "[y] Execute",
		"tui.opt_cancel":     "[n] Cancel",
		"tui.opt_explain":    "[e] Explain",
		"tui.cancelled":      "Cancelled.",
		"tui.explain_prefix": "\nExplanation: %s\n\n",
		"tui.run_error":      "TUI error: %v",

		// Guardrails — warning box
		"guard.warning_title":  "CRITICAL SAFETY WARNING: Dangerous command detected!",
		"guard.warning_desc":   "This command may cause irreversible damage to your system!",
		"guard.warning_prompt": "Please verify carefully, then type YES to continue.",
		"guard.warning_line1":  "╔══════════════════════════════════════════════════════════════╗",
		"guard.warning_line2":  "║  ⚠  %-52s ║",
		"guard.warning_line3":  "╠══════════════════════════════════════════════════════════════╣",
		"guard.warning_line4":  "║  Command: %-45s ║",
		"guard.warning_line5":  "║  %-56s ║",
		"guard.warning_line6":  "╚══════════════════════════════════════════════════════════════╝",

		// Guardrails — YES confirmation input
		"guard.confirm_title":       "Type YES to confirm execution of this dangerous command",
		"guard.confirm_prompt":      "> ",
		"guard.confirm_cancel_hint": "Press Esc to cancel",
		"guard.confirm_validate":    "Please type YES (uppercase) to confirm",
		"guard.confirm_cancelled":   "Dangerous command cancelled.",

		// Config display
		"config.title":      "Current configuration (~/.shell-mate.yaml):",
		"config.api_key":    "  API_KEY        : %s",
		"config.api_base":   "  API_BASE_URL   : %s",
		"config.model_name": "  MODEL_NAME     : %s",
		"config.search_key": "  SEARCH_API_KEY : %s",
		"config.language":   "  LANGUAGE       : %s",
		"config.danger_list": "  DANGER_KEYWORDS: %v",
		"config.unset":      "(not set)",
		"config.saved":      "Configuration saved to ~/.shell-mate.yaml",
		"config.write_err":  "Failed to write config: %s",
		"config.read_err":   "Error reading config: %s",

		// root.go
		"root.config_missing": "Error: Please set API_KEY first, run: sm config -k <your-api-key>",
		"root.llm_fail":       "AI call failed: %v",
		"root.home_err":       "Cannot get user home directory:",
		"root.llm_nocmd":      "AI could not generate a valid command. Try rephrasing your request.",

		// Command execution
		"exec.run_error": "Command execution failed: %v",

		// Spinner messages (for spinning animation)
		"root.llm_calling":     "Requesting AI translation...",
		"root.search_spin":     "Complex request, searching the web...",
		"root.search_spin_done": "Search done, regenerating command...",

		// Agentic search (Phase 5 slow path)
		"root.search_start":    "\n[i] Complex request detected, searching the web for solutions...",
		"root.search_done":     "[✓] Search complete, regenerating command based on search results...\n",
		"root.search_fail":     "[✗] Web search failed: %v\nFalling back to LLM's direct answer.",
		"root.search_fallback": "[i] No search results found, using LLM direct answer.",
		"root.search_still":    "[!] LLM still uncertain after search, here is the best attempt:",
	},
}

// t 根据当前语言设置返回翻译后的字符串
func t(key string) string {
	lang := getCurrentLang()
	if msg, ok := messages[lang][key]; ok {
		return msg
	}
	// 回退到中文
	if msg, ok := messages[LangZH][key]; ok {
		return msg
	}
	return key
}

// getCurrentLang 从 viper 配置中读取当前语言，默认中文
func getCurrentLang() Lang {
	lang := viper.GetString("language")
	switch Lang(lang) {
	case LangZH, LangEN:
		return Lang(lang)
	}
	return LangZH
}

// CurrentLang 供外部包获取当前语言
func CurrentLang() string {
	return string(getCurrentLang())
}
