package llm

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// GatherContext 收集当前系统环境信息，作为 LLM 提示的一部分
func GatherContext() string {
	var sb strings.Builder

	// 操作系统类型（如 windows, linux, darwin）
	sb.WriteString(fmt.Sprintf("操作系统: %s\n", runtime.GOOS))

	// 当前使用的 Shell 程序
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "unknown"
	}
	sb.WriteString(fmt.Sprintf("当前Shell: %s\n", shell))

	// 当前工作目录下的文件列表（仅文件名，不读取内容）
	sb.WriteString("当前目录文件列表:\n")
	entries, err := os.ReadDir(".")
	if err != nil {
		sb.WriteString(fmt.Sprintf("(无法读取目录: %v)\n", err))
	} else {
		for _, entry := range entries {
			sb.WriteString(fmt.Sprintf("  - %s", entry.Name()))
			if entry.IsDir() {
				sb.WriteString("/")
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
