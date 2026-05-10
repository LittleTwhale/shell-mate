package llm

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// contextLabels 按语言存储上下文标签
var contextLabels = map[string]struct {
	osLabel   string // "操作系统: %s\n" / "OS: %s\n"
	shellLabel  string // "当前Shell: %s\n" / "Shell: %s\n"
	dirLabel    string // "当前目录文件列表:\n" / "Directory listing:\n"
	dirErrLabel string // "(无法读取目录: %v)\n" / "(cannot read directory: %v)\n"
	dirItemFmt  string // "  - %s" (same in both languages)
}{
	"zh": {
		osLabel:     "操作系统: %s\n",
		shellLabel:  "当前Shell: %s\n",
		dirLabel:    "当前目录文件列表:\n",
		dirErrLabel: "(无法读取目录: %v)\n",
		dirItemFmt:  "  - %s",
	},
	"en": {
		osLabel:     "OS: %s\n",
		shellLabel:  "Shell: %s\n",
		dirLabel:    "Directory listing:\n",
		dirErrLabel: "(cannot read directory: %v)\n",
		dirItemFmt:  "  - %s",
	},
}

// getContextLabels 根据语言获取上下文标签，默认为中文
func getContextLabels(lang string) struct {
	osLabel   string
	shellLabel  string
	dirLabel    string
	dirErrLabel string
	dirItemFmt  string
} {
	if l, ok := contextLabels[lang]; ok {
		return l
	}
	return contextLabels["zh"]
}

// GatherContext 收集当前系统环境信息，作为 LLM 提示的一部分
// lang: 当前语言 (zh/en)，影响标签文本的语言
func GatherContext(lang string) string {
	labels := getContextLabels(lang)
	var sb strings.Builder

	// 操作系统类型（如 windows, linux, darwin）
	sb.WriteString(fmt.Sprintf(labels.osLabel, runtime.GOOS))

	// 当前使用的 Shell 程序
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "unknown"
	}
	sb.WriteString(fmt.Sprintf(labels.shellLabel, shell))

	// 当前工作目录下的文件列表（仅文件名，不读取内容）
	sb.WriteString(labels.dirLabel)
	entries, err := os.ReadDir(".")
	if err != nil {
		sb.WriteString(fmt.Sprintf(labels.dirErrLabel, err))
	} else {
		for _, entry := range entries {
			sb.WriteString(fmt.Sprintf(labels.dirItemFmt, entry.Name()))
			if entry.IsDir() {
				sb.WriteString("/")
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
