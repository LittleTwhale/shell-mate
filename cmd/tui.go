package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/charmbracelet/huh"

	"shell-mate/llm"
)

// TUIResult showTUI 的返回结果，告知调用方下一步动作
type TUIResult struct {
	Action string // "executed" / "cancelled" / "retry"
	Stderr string // 命令执行失败时的错误输出，仅 retry 时有效
}

// showTUI 显示交互式命令确认菜单
// 用户可选择执行/取消/解释/学习。执行失败后提供 AI 修正重试选项。
// dangerous 为 true 时，执行前需要输入 YES 二次确认
func showTUI(provider llm.Provider, cmdStr, explain string, dangerous bool) TUIResult {
	for {
		var choice string

		err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title(t("tui.title")).
					Description(fmt.Sprintf(t("tui.description"), cmdStr)).
					Options(
						huh.NewOption(t("tui.opt_execute"), "y"),
						huh.NewOption(t("tui.opt_cancel"), "n"),
						huh.NewOption(t("tui.opt_explain"), "e"),
						huh.NewOption(t("tui.opt_learn"), "l"),
					).
					Value(&choice),
			),
		).Run()

		if err != nil {
			fmt.Fprintf(os.Stderr, t("tui.run_error")+"\n", err)
			os.Exit(1)
		}

		switch choice {
		case "y":
			// 高危命令需要输入 YES 确认
			if dangerous {
				if !confirmDangerous() {
					continue // 未输入 YES，返回菜单重新选择
				}
			}
			// 执行命令，捕获 stderr
			exitCode, stderr, execErr := executeCommand(cmdStr)
			if execErr != nil || exitCode != 0 {
				// 命令执行失败，展示 AI 修正重试菜单
				action := showRetryMenu(cmdStr, stderr)
				return TUIResult{Action: action, Stderr: stderr}
			}
			// 执行成功
			return TUIResult{Action: "executed"}

		case "n":
			fmt.Println(t("tui.cancelled"))
			return TUIResult{Action: "cancelled"}

		case "e":
			fmt.Printf(t("tui.explain_prefix"), explain)
			// 继续循环，重新展示菜单

		case "l":
			// 调用 LLM 生成命令学习卡片（返回 Markdown 纯文本）
			lang := CurrentLang()
			sp := startSpinner(fmt.Sprintf(t("learn.spinner"), cmdStr))
			cardContent, learnErr := llm.CallLLMForLearning(provider, cmdStr, lang)
			sp.stop("")
			if learnErr != nil {
				fmt.Fprintf(os.Stderr, t("learn.fail")+"\n", learnErr)
			} else {
				printKnowledgeCard(cmdStr, cardContent)
			}

			// 防止 TUI 菜单遮挡长文本或错误信息，等待用户阅读完并回车后再刷新
			fmt.Println("\n按回车键继续...")
			bufio.NewReader(os.Stdin).ReadBytes('\n')

			// 继续循环，重新展示菜单
		}
	}
}

// showRetryMenu 命令执行失败后展示 AI 修正重试菜单
func showRetryMenu(failedCmd, stderr string) string {
	// 截断过长的错误输出，仅保留前 200 字符用于显示
	stderrDisplay := strings.TrimSpace(stderr)
	if len(stderrDisplay) > 200 {
		stderrDisplay = stderrDisplay[:200] + "..."
	}
	if stderrDisplay == "" {
		stderrDisplay = "(无错误输出)"
	}

	fmt.Println()
	fmt.Fprintf(os.Stderr, "\033[1;33m%s\033[0m\n", t("tui.retry_title"))
	fmt.Fprintf(os.Stderr, "\033[90m%s\033[0m\n", stderrDisplay)

	var choice string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(t("tui.retry_title")).
				Description(fmt.Sprintf(t("tui.retry_description"), stderrDisplay)).
				Options(
					huh.NewOption(t("tui.retry_opt_ai"), "r"),
					huh.NewOption(t("tui.retry_opt_cancel"), "n"),
				).
				Value(&choice),
		),
	).Run()

	if err != nil {
		fmt.Fprintln(os.Stderr, t("tui.retry_cancelled"))
		return "cancelled"
	}

	switch choice {
	case "r":
		return "retry"
	default:
		fmt.Fprintln(os.Stderr, t("tui.retry_cancelled"))
		return "cancelled"
	}
}

// executeCommand 执行 Shell 命令，实时输出到终端，同时捕获 stderr
// 返回值: exitCode 进程退出码（0 表示成功），stderr 捕获的错误输出，err 执行层面的错误
func executeCommand(cmdStr string) (exitCode int, stderr string, err error) {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		if os.Getenv("PSModulePath") != "" {
			cmd = exec.Command("powershell", "-NoProfile", "-Command", cmdStr)
		} else {
			cmd = exec.Command("cmd", "/c", cmdStr)
		}
	} else {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
		cmd = exec.Command(shell, "-c", cmdStr)
	}

	// stderr 实时输出到终端，同时写入缓冲区供纠错使用
	var stderrBuf bytes.Buffer
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	fmt.Println()
	runErr := cmd.Run()

	// 获取退出码
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// 其他执行错误（如找不到命令）
			exitCode = -1
			err = runErr
		}
	}

	// 如果有非退出码类型的错误，把错误信息也加入 stderr
	if err != nil {
		stderrBuf.WriteString(err.Error())
	}

	return exitCode, stderrBuf.String(), runErr
}

// fillPlaceholders 扫描命令中的 <...> 占位符，如果存在，则弹窗让用户填空
func fillPlaceholders(cmdStr string) (string, error) {
	// 匹配形如 <Server_IP>, <File_Path> 的占位符
	re := regexp.MustCompile(`<([^>]+)>`)
	matches := re.FindAllStringSubmatch(cmdStr, -1)

	if len(matches) == 0 {
		return cmdStr, nil // 没有占位符，直接返回原命令
	}

	// 1. 提取不重复的占位符（防止同一个变量在命令中出现多次，比如 <IP> 出现两次只需输入一次）
	uniquePlaceholders := make(map[string]bool)
	var placeholders []string
	for _, m := range matches {
		key := m[1]
		if !uniquePlaceholders[key] {
			uniquePlaceholders[key] = true
			placeholders = append(placeholders, key)
		}
	}

	// 2. 动态构造 huh 表单字段
	var inputs []huh.Field
	answers := make(map[string]*string)

	for _, p := range placeholders {
		val := new(string)
		answers[p] = val
		inputs = append(inputs, huh.NewInput().
			Title(fmt.Sprintf(t("tui.placeholder_prompt"), p)).
			Value(val))
	}

	// 3. 运行填空表单
	form := huh.NewForm(
		huh.NewGroup(inputs...).
			Title(t("tui.placeholder_title")).
			Description(fmt.Sprintf(t("tui.placeholder_desc"), cmdStr)),
	)

	err := form.Run()
	if err != nil {
		return "", err // 用户按 Esc 取消
	}

	// 4. 将用户输入替换回原命令
	filledCmd := cmdStr
	for _, p := range placeholders {
		filledCmd = strings.ReplaceAll(filledCmd, "<"+p+">", *answers[p])
	}

	return filledCmd, nil
}
