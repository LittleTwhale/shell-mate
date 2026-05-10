package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"shell-mate/llm"
	"shell-mate/search"
)

var cfgFile string

// rootCmd 是 shell-mate 的根命令，接收自然语言描述并翻译为 Shell 命令
var rootCmd = &cobra.Command{
	Use:   "sm [自然语言描述]",
	Short: "shell-mate — 终端命令行 AI 助手",
	Long: `shell-mate 是一个基于 Go 开发的终端命令行 AI 助手，
能将你的自然语言请求即时翻译成可执行的 Shell 命令。`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cobraCmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cobraCmd.Help()
			return
		}

		// 获取当前语言设置
		lang := CurrentLang()

		// 读取 API 密钥，兼容旧的 openai_api_key 配置项
		apiKey := viper.GetString("api_key")
		if apiKey == "" {
			apiKey = viper.GetString("openai_api_key")
		}
		if apiKey == "" {
			fmt.Fprintln(os.Stderr, t("root.config_missing"))
			os.Exit(1)
		}

		// 读取 API 端点地址和模型名称
		apiBaseURL := viper.GetString("api_base_url")
		modelName := viper.GetString("model_name")

		// 收集系统上下文
		context := llm.GatherContext(lang)

		// 第一次 LLM 调用 — 启动旋转动画提供可视化反馈
		sp := startSpinner(fmt.Sprintf("%s %s", t("root.llm_calling"), args[0]))
		resp, err := llm.CallLLM(apiBaseURL, apiKey, modelName, context, args[0], lang)
		if err != nil {
			sp.stop("")
			fmt.Fprintf(os.Stderr, t("root.llm_fail")+"\n", err)
			os.Exit(1)
		}

		// 保存原始响应作为回退（当 need_search 为 true 但仍可能有部分命令时）
		fallbackResp := resp

		// 慢路径：LLM 表示需要联网搜索（仅需 need_search 为 true 即触发）
		if resp.NeedSearch {
			// 更新 spinner 为搜索状态
			sp.update(t("root.search_spin"))

			// 使用用户原始描述作为关键词调用搜索 API
			searchResults, searchErr := search.Search(args[0], 3)
			if searchErr != nil {
				sp.stop("")
				fmt.Fprintf(os.Stderr, t("root.search_fail")+"\n", searchErr)
				// 搜索失败时使用第一次调用的结果作为回退
				if resp.Cmd == "" {
					fmt.Println(t("root.search_fallback"))
				} else {
					// 有 cmd 就继续用，但提示搜索失败
					fmt.Fprintln(os.Stderr, t("root.search_fallback"))
				}
			}

			// 搜索成功时，基于搜索结果发起第二次 LLM 调用
			if searchResults != nil && len(searchResults) > 0 {
				sp.update(t("root.search_spin_done"))
				flatResults := search.FlattenResults(searchResults, lang)
				resp2, err2 := llm.CallLLMWithSearch(apiBaseURL, apiKey, modelName, context, args[0], flatResults, lang)
				if err2 != nil {
					sp.stop("")
					fmt.Fprintf(os.Stderr, t("root.llm_fail")+"\n", err2)
					os.Exit(1)
				}

				// 如果二次调用后仍标记 need_search，给用户一个提示
				if resp2.NeedSearch {
					sp.stop("")
					fmt.Println(t("root.search_still"))
				}
				resp = resp2
			} else {
				// 搜索无结果，使用第一次调用结果
				resp = fallbackResp
			}
		}

		// 停止旋转动画
		sp.stop("")

		// 最终检查：LLM 未能生成有效命令时退出
		if resp.Cmd == "" {
			fmt.Fprintln(os.Stderr, t("root.llm_nocmd"))
			os.Exit(1)
		}

		// 检查命令是否包含高危关键词，在展示菜单前进行安全扫描
		dangerous := isDangerous(resp.Cmd)
		if dangerous {
			printDangerWarning(resp.Cmd)
		}

		// 调用交互式 TUI 菜单，让用户选择执行/取消/解释
		showTUI(resp.Cmd, resp.Explain, dangerous)
	},
}

// Execute 执行根命令，由 main.go 调用
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

// initConfig 初始化 viper 配置，读取 ~/.shell-mate.yaml
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, t("root.home_err"), err)
			os.Exit(1)
		}
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".shell-mate")
	}

	// 设置默认值
	viper.SetDefault("api_base_url", "https://api.deepseek.com")
	viper.SetDefault("model_name", "deepseek-v4-flash")
	viper.SetDefault("language", "zh")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, t("config.read_err")+"\n", err)
		}
	}

	// 根据语言设置动态更新 Cobra 命令元数据
	rootCmd.Use = t("root.use")
	rootCmd.Short = t("root.short")
	rootCmd.Long = t("root.long")

	configCmd.Short = t("config.short")
	configCmd.Long = t("config.long")
}
