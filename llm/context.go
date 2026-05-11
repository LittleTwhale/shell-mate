package llm

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// contextLabels 按语言存储上下文标签
var contextLabels = map[string]struct {
	osLabel     string // "操作系统: %s\n" / "OS: %s\n"
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
	osLabel     string
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

// isFileRelated 启发式判断用户的自然语言需求是否涉及文件系统
func isFileRelated(query string) bool {
	// 常见的文件操作、目录操作关键词
	keywords := []string{
		"文件", "目录", "夹", "路径", "file", "dir", "folder", "path",
		"ls", "find", "grep", "cat", "rm", "mv", "cp", "tar", "zip", "unzip", "awk", "sed",
	}
	queryLower := strings.ToLower(query)
	for _, kw := range keywords {
		if strings.Contains(queryLower, kw) {
			return true
		}
	}
	return false
}

// GatherContext 收集当前系统环境信息，作为 LLM 提示的一部分
// lang: 当前语言 (zh/en)，影响标签文本的语言
func GatherContext(lang string, userQuery string) string {
	labels := getContextLabels(lang)
	var sb strings.Builder

	// 操作系统类型（如 windows, linux, darwin）
	sb.WriteString(fmt.Sprintf(labels.osLabel, runtime.GOOS))

	// 当前使用的 Shell 程序
	shell := os.Getenv("SHELL")
	if shell == "" {
		if runtime.GOOS == "windows" {
			// 启发式探测：PowerShell 环境通常带有 PSModulePath 变量
			if os.Getenv("PSModulePath") != "" {
				shell = "powershell"
			} else {
				shell = "cmd"
			}
		} else {
			shell = "unknown"
		}
	}
	sb.WriteString(fmt.Sprintf("当前Shell: %s\n", shell))

	// 当前工作目录下的文件列表（仅文件名，不读取内容,惰性加载）
	if isFileRelated(userQuery) {
		sb.WriteString(labels.dirLabel)
		entries, err := os.ReadDir(".")
		if err != nil {
			sb.WriteString(fmt.Sprintf(labels.dirErrLabel, err))
		} else {
			const maxFiles = 30 // 最大读取数量（瘦身截断）
			count := 0
			
			for _, entry := range entries {
				name := entry.Name()
				// 过滤掉极其庞大且对生成命令往往无用的黑洞目录
				if name == ".git" || name == "node_modules" || name == "vendor" || name == "__pycache__" {
					continue
				}

				sb.WriteString(fmt.Sprintf(labels.dirItemFmt, name))
				if entry.IsDir() {
					sb.WriteString("/")
				}
				sb.WriteString("\n")

				count++
				if count >= maxFiles {
					// 达到上限，添加省略号提示并终止
					if lang == "en" {
						sb.WriteString("  ... (truncated for brevity)\n")
					} else {
						sb.WriteString("  ... (已截断更多文件)\n")
					}
					break
				}
			}
		}
	}

	return sb.String()
}
