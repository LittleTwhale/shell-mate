package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
)

// dangerousKeywords 高危命令关键词黑名单
// 任何包含这些关键词的命令都需要额外的 YES 确认才能执行
var dangerousKeywords = []string{
	"rm -rf",
	"rm -r ",
	"rm -fr",
	"mkfs",
	"> /dev/sda",
	"> /dev/sdb",
	"> /dev/sdc",
	"> /dev/sdd",
	"> /dev/nvme",
	"dd if=",
	"chmod 777",
	"chmod -R 777",
	"chmod -R 7777",
	"chown -R",
	":(){ :|:& };:",
	"> /dev/sd",
	"> /dev/hd",
	"wget ",
	"curl ",
	"| sh",
	"| bash",
	"fork bomb",
	"shutdown",
	"reboot",
	"init 0",
	"init 6",
	"del /f /s",
	"format c:",
	"diskpart",
}

// isDangerous 扫描命令是否包含高危关键词
func isDangerous(cmd string) bool {
	cmdLower := strings.ToLower(cmd)
	for _, kw := range dangerousKeywords {
		if strings.Contains(cmdLower, kw) {
			return true
		}
	}
	return false
}

// printDangerWarning 在终端打印醒目的红色高危警告
func printDangerWarning(cmd string) {
	red := "\033[1;31m"
	reset := "\033[0m"

	fmt.Println()
	fmt.Println(red + "╔══════════════════════════════════════════════════════════════╗" + reset)
	fmt.Println(red + "║  ⚠  严重安全警告：检测到高危命令操作！                     ║" + reset)
	fmt.Println(red + "╠══════════════════════════════════════════════════════════════╣" + reset)
	if len(cmd) > 56 {
		cmd = cmd[:56]
	}
	fmt.Printf(red+"║  命令: %-52s ║\n"+reset, cmd)
	fmt.Println(red + "║  该命令可能对您的系统造成不可逆的损害！                     ║" + reset)
	fmt.Println(red + "║  请仔细确认无误后，输入 YES 以继续。                       ║" + reset)
	fmt.Println(red + "╚══════════════════════════════════════════════════════════════╝" + reset)
	fmt.Println()
}

// confirmDangerous 弹出一个文本输入框，强制用户输入大写 YES 才能放行
// 返回 true 表示用户成功输入了 YES
func confirmDangerous() bool {
	var confirm string

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("请输入大写的 YES 以确认执行此高危命令").
				Prompt("> ").
				Validate(func(s string) error {
					if s != "YES" {
						return fmt.Errorf("请输入 YES（大写）以确认")
					}
					return nil
				}).
				Value(&confirm),
		),
	).Run()

	if err != nil {
		fmt.Println("已取消。")
		return false
	}
	return confirm == "YES"
}
