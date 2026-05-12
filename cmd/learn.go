package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"shell-mate/llm"
)

// learnCmd sm learn 子命令 — 为任意 Shell 命令生成知识卡片
var learnCmd = &cobra.Command{
	Use:   "learn [命令]",
	Short: "为指定命令生成知识卡片",
	Long:  `为指定的 Shell 命令生成一张详细的"知识卡片"，包括命令概览、涉及的工具、参数详解、常见变体、最佳实践和注意事项。`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cobraCmd *cobra.Command, args []string) {
		cmdStr := strings.Join(args, " ")
		lang := CurrentLang()

		// 读取配置
		apiKey := viper.GetString("api_key")
		if apiKey == "" {
			apiKey = viper.GetString("openai_api_key")
		}
		if apiKey == "" {
			fmt.Fprintln(os.Stderr, t("root.config_missing"))
			os.Exit(1)
		}

		apiBaseURL := viper.GetString("api_base_url")
		modelName := viper.GetString("model_name")
		providerName := viper.GetString("provider")
		if providerName == "" {
			providerName = "deepseek"
		}

		provider, err := llm.NewProvider(providerName, apiBaseURL, apiKey, modelName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "创建 LLM Provider 失败: %v\n", err)
			os.Exit(1)
		}

		// 生成学习卡片（返回 Markdown 纯文本）
		sp := startSpinner(fmt.Sprintf(t("learn.spinner"), cmdStr))
		cardContent, err := llm.CallLLMForLearning(provider, cmdStr, lang)
		sp.stop("")

		if err != nil {
			fmt.Fprintf(os.Stderr, t("learn.fail")+"\n", err)
			os.Exit(1)
		}

		printKnowledgeCard(cmdStr, cardContent)
	},
}

func init() {
	rootCmd.AddCommand(learnCmd)
}

// printKnowledgeCard 在终端中渲染知识卡片，带彩色边框和 Markdown 语法高亮
// 优先使用 Glamour 渲染 Markdown，并用 Lipgloss 包裹美观的边框
func printKnowledgeCard(cmdStr, cardContent string) {
	// 1. 定义 Lipgloss 样式 (这就相当于 Python rich 中的 Panel)
	// 你可以自由修改这里的 Color("xx") 来换颜色，支持 ANSI 256 色或 Hex 颜色值
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).       // 圆角边框
		BorderForeground(lipgloss.Color("63")). // 边框颜色 (优雅的紫色)
		Padding(1, 2).                          // 上下内边距1，左右内边距2
		MarginTop(1).                           // 上外边距
		MarginBottom(1).                        // 下外边距
		Width(80)                               // 卡片整体宽度

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")) // 粉红色标题
	subHeaderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))         // 灰色原命令
	dividerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))           // 分割线颜色

	// 2. 截断过长的命令以防破坏排版
	displayCmd := cmdStr
	if len(displayCmd) > 66 {
		displayCmd = displayCmd[:66] + "..."
	}

	// 3. 配置 Glamour 渲染器
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(74), // 宽度设为 74，给 Lipgloss 的 Padding 留出空间
	)

	// 4. 尝试渲染并组装最终的 UI
	if err == nil {
		renderedContent, renderErr := r.Render(cardContent)
		if renderErr == nil {
			// 组装头部信息
			header := headerStyle.Render("📚 命令学习卡片")
			subHeader := subHeaderStyle.Render(fmt.Sprintf("原命令: %s", displayCmd))
			divider := dividerStyle.Render(strings.Repeat("─", 74))

			// 将头部信息和 Glamour 渲染的 Markdown 内容拼接
			fullContent := fmt.Sprintf("%s\n%s\n%s\n\n%s", header, subHeader, divider, strings.TrimSpace(renderedContent))

			// 用 Lipgloss 卡片样式包裹并打印
			fmt.Println(cardStyle.Render(fullContent))
			return
		}
	}

	// 如果报错，走兜底方案
	fallbackPrintKnowledgeCard(cmdStr, cardContent)
}

// fallbackPrintKnowledgeCard 修正后的手动 Markdown 解析器（兜底方案）
func fallbackPrintKnowledgeCard(cmdStr, cardContent string) {
	// 打印头部卡片框 (兜底时使用朴素的 ASCII)
	displayCmd := cmdStr
	if len(displayCmd) > 66 {
		displayCmd = displayCmd[:66] + "..."
	}
	fmt.Println()
	fmt.Printf("╭──────────────────────────────────────────────────────────────────────────────╮\n")
	fmt.Printf("│  📚 %-70s │\n", t("learn.title"))
	fmt.Printf("├──────────────────────────────────────────────────────────────────────────────┤\n")
	fmt.Printf("│  %s: %-67s │\n", t("learn.original_cmd"), displayCmd)
	fmt.Printf("╰──────────────────────────────────────────────────────────────────────────────╯\n")
	fmt.Println()

	const (
		colorReset  = "\033[0m"
		colorCyan   = "\033[1;36m"
		colorYellow = "\033[1;33m"
		colorGreen  = "\033[1;32m"
		colorBlue   = "\033[0;34m"
		colorBold   = "\033[1m"
		colorDim    = "\033[2m"
	)

	lines := strings.Split(cardContent, "\n")
	inCodeBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			if inCodeBlock {
				langLabel := strings.TrimPrefix(trimmed, "```")
				langLabel = strings.TrimSpace(langLabel)
				if langLabel != "" {
					fmt.Printf("%s┌─ %s%s\n", colorDim, langLabel, colorReset)
				} else {
					fmt.Printf("%s┌────%s\n", colorDim, colorReset)
				}
			} else {
				fmt.Printf("%s└────%s\n", colorDim, colorReset)
			}
			continue
		}

		if inCodeBlock {
			fmt.Printf("%s│ %s%s\n", colorBlue, line, colorReset)
			continue
		}

		if strings.HasPrefix(trimmed, "## ") {
			title := strings.TrimPrefix(trimmed, "## ")
			fmt.Printf("\n%s%s%s\n", colorCyan, title, colorReset)
			continue
		}

		if strings.HasPrefix(trimmed, "### ") {
			title := strings.TrimPrefix(trimmed, "### ")
			fmt.Printf("\n%s%s%s\n", colorYellow, title, colorReset)
			continue
		}

		if strings.HasPrefix(trimmed, "- ") {
			line = strings.Replace(line, "- ", "  • ", 1)
		} else if strings.HasPrefix(trimmed, "* ") {
			line = strings.Replace(line, "* ", "  • ", 1)
		}

		rendered := renderInlineMarkdown(line, colorGreen, colorBold, colorReset)
		fmt.Println(rendered)
	}
	fmt.Println()
}

// renderInlineMarkdown 渲染单行中的内联 Markdown 语法：`代码` 和 **粗体**
func renderInlineMarkdown(line, codeColor, boldColor, reset string) string {
	// 渲染内联代码 `...`
	result := line
	for {
		start := strings.Index(result, "`")
		if start == -1 {
			break
		}
		end := strings.Index(result[start+1:], "`")
		if end == -1 {
			break
		}
		end += start + 1
		// 替换 `code` 为彩色版本
		codeText := result[start+1 : end]
		result = result[:start] + codeColor + codeText + reset + result[end+1:]
	}

	// 渲染粗体 **...**
	for {
		start := strings.Index(result, "**")
		if start == -1 {
			break
		}
		end := strings.Index(result[start+2:], "**")
		if end == -1 {
			break
		}
		end += start + 2
		boldText := result[start+2 : end]
		result = result[:start] + boldColor + boldText + reset + result[end+2:]
	}

	return result
}
