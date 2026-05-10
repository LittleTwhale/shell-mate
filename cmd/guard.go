package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/viper"
)

// defaultDangerousKeywords 默认的高危命令关键词黑名单
// 任何包含这些关键词的命令都需要额外的 YES 确认才能执行
var defaultDangerousKeywords = []string{
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

// getDangerousKeywords 获取当前生效的高危关键词列表
func getDangerousKeywords() []string {
	customKeywords := viper.GetStringSlice("dangerous_keywords")
	
	// 如果用户没有配置自定义关键词，直接返回默认列表
	if len(customKeywords) == 0 {
		return defaultDangerousKeywords
	}

	// 合并默认列表和自定义列表
	var merged []string
	merged = append(merged, defaultDangerousKeywords...)
	merged = append(merged, customKeywords...)
	
	return merged
}

// isDangerous 扫描命令是否包含高危关键词
func isDangerous(cmd string) bool {
	cmdLower := strings.ToLower(cmd)
	for _, kw := range getDangerousKeywords() {
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

	truncated := cmd
	if len(truncated) > 48 {
		truncated = truncated[:48]
	}

	fmt.Println()
	fmt.Println(red + t("guard.warning_line1") + reset)
	fmt.Printf(red+t("guard.warning_line2")+"\n"+reset, t("guard.warning_title"))
	fmt.Println(red + t("guard.warning_line3") + reset)
	fmt.Printf(red+t("guard.warning_line4")+"\n"+reset, truncated)
	fmt.Printf(red+t("guard.warning_line5")+"\n"+reset, t("guard.warning_desc"))
	fmt.Printf(red+t("guard.warning_line5")+"\n"+reset, t("guard.warning_prompt"))
	fmt.Println(red + t("guard.warning_line6") + reset)
	fmt.Println()
}

// confirmDangerous 弹出一个文本输入框，强制用户输入大写 YES 才能放行
// 用户可按 Esc 键取消，返回 true 表示用户成功输入了 YES
func confirmDangerous() bool {
	var confirm string

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(t("guard.confirm_title")).
				Description(t("guard.confirm_cancel_hint")).
				Prompt(t("guard.confirm_prompt")).
				Validate(func(s string) error {
					if s != "YES" {
						return fmt.Errorf("%s", t("guard.confirm_validate"))
					}
					return nil
				}).
				Value(&confirm),
		),
	).Run()

	if err != nil {
		// 用户按下了 Esc 或其他中断操作
		fmt.Println(t("guard.confirm_cancelled"))
		return false
	}
	return confirm == "YES"
}
