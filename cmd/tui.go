package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/charmbracelet/huh"
)

// showTUI 显示交互式选择菜单，让用户确认是否执行命令
// 选项 e（解释）会打印解释后重新展示菜单，形成循环
// dangerous 为 true 时，选择 [y] 不会直接执行，而是弹出 YES 确认输入框
func showTUI(cmdStr, explain string, dangerous bool) {
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
					).
					Value(&choice),
			),
		).Run()

		if err != nil {
			fmt.Fprintf(os.Stderr, "TUI 运行失败: %v\n", err)
			os.Exit(1)
		}

		switch choice {
		case "y":
			// 高危命令需要输入 YES 确认，禁止直接执行
			if dangerous {
				if !confirmDangerous() {
					continue // 用户未输入 YES，返回菜单重新选择
				}
			}
			executeCommand(cmdStr)
			return
		case "n":
			fmt.Println(t("tui.cancelled"))
			os.Exit(0)
		case "e":
			fmt.Printf(t("tui.explain_prefix"), explain)
			// 继续循环，重新展示菜单
		}
	}
}

// executeCommand 执行 Shell 命令，并将 Stdout/Stderr 实时重定向到终端
// 在 Windows 上使用 cmd /c，在 Unix 上使用 $SHELL -c
func executeCommand(cmdStr string) {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", cmdStr)
	} else {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
		cmd = exec.Command(shell, "-c", cmdStr)
	}

	// 将命令的输入输出直接连接到终端，确保实时交互
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	fmt.Println()
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "\n命令执行失败: %v\n", err)
	}
}
