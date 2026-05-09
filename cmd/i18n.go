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
		// TUI 交互菜单
		"tui.title":          "是否执行此命令?",
		"tui.description":    "命令: %s",
		"tui.opt_execute":    "[y] 执行 (Execute)",
		"tui.opt_cancel":     "[n] 取消 (Cancel)",
		"tui.opt_explain":    "[e] 解释 (Explain)",
		"tui.cancelled":      "已取消。",
		"tui.explain_prefix": "\n命令解释: %s\n\n",

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
		"config.unset":      "(未设置)",
		"config.saved":      "配置已保存到 ~/.shell-mate.yaml",
		"config.write_err":  "写入配置文件失败: %s",
		"config.read_err":   "读取配置文件出错: %s",

		// root.go
		"root.config_missing": "错误: 请先设置 API_KEY，运行: sm config -k <your-api-key>",
		"root.llm_fail":       "调用 AI 失败: %v",
		"root.search_future":  "联网搜索功能将在后续版本中开放。",
		"root.home_err":       "无法获取用户主目录:",
	},
	LangEN: {
		// TUI interactive menu
		"tui.title":          "Execute this command?",
		"tui.description":    "Command: %s",
		"tui.opt_execute":    "[y] Execute",
		"tui.opt_cancel":     "[n] Cancel",
		"tui.opt_explain":    "[e] Explain",
		"tui.cancelled":      "Cancelled.",
		"tui.explain_prefix": "\nExplanation: %s\n\n",

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
		"config.unset":      "(not set)",
		"config.saved":      "Configuration saved to ~/.shell-mate.yaml",
		"config.write_err":  "Failed to write config: %s",
		"config.read_err":   "Error reading config: %s",

		// root.go
		"root.config_missing": "Error: Please set API_KEY first, run: sm config -k <your-api-key>",
		"root.llm_fail":       "AI call failed: %v",
		"root.search_future":  "Web search will be available in a future version.",
		"root.home_err":       "Cannot get user home directory:",
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
