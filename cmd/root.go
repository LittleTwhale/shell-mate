package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"shell-mate/llm"
	"shell-mate/search"
)

var (
	cfgFile  string
	fastMode bool // 极速模式标志位
)

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

		lang := CurrentLang()

		// 读取 API 密钥
		apiKey := viper.GetString("api_key")
		if apiKey == "" {
			apiKey = viper.GetString("openai_api_key")
		}
		if apiKey == "" {
			fmt.Fprintln(os.Stderr, t("root.config_missing"))
			os.Exit(1)
		}

		// 读取 Provider 配置
		apiBaseURL := viper.GetString("api_base_url")
		modelName := viper.GetString("model_name")
		providerName := viper.GetString("provider")
		if providerName == "" {
			providerName = "deepseek"
		}

		// 创建 LLM Provider 实例
		provider, err := llm.NewProvider(providerName, apiBaseURL, apiKey, modelName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "创建 LLM Provider 失败: %v\n", err)
			os.Exit(1)
		}

		// 先提取用户查询
		userQuery := args[0]

		// 将 userQuery 传给 GatherContext，让它决定要不要读取目录
		context := llm.GatherContext(lang, userQuery)

		// 初始 LLM 翻译
		var resp *llm.LLMResponse
		var llmErr error

		resp, llmErr = initialCall(provider, context, userQuery, lang, fastMode)
		if llmErr != nil {
			fmt.Fprintf(os.Stderr, t("root.llm_fail")+"\n", llmErr)
			os.Exit(1)
		}

		// 主循环：展示命令 → 执行 → 可能失败 → AI 修正 → 重新展示
		for {
			// 最终检查：LLM 未能生成有效命令
			if resp.Cmd == "" {
				fmt.Fprintln(os.Stderr, t("root.llm_nocmd"))
				os.Exit(1)
			}

			// 填空题交互拦截
			filledCmd, fillErr := fillPlaceholders(resp.Cmd)
			if fillErr != nil {
				// 用户在填空表单处按下了 Esc 取消
				fmt.Println(t("tui.cancelled"))
				return
			}
			// 将替换完成的命令赋值回去
			resp.Cmd = filledCmd

			// 检查命令是否包含高危关键词
			dangerous := isDangerous(resp.Cmd)
			if dangerous {
				printDangerWarning(resp.Cmd)
			}

			// 展示 TUI 菜单，用户选择执行/取消/解释
			result := showTUI(resp.Cmd, resp.Explain, dangerous)

			switch result.Action {
			case "executed":
				// 命令执行成功，退出
				return

			case "cancelled":
				// 用户取消，退出
				return

			case "retry":
				// 命令执行失败，请求 AI 修正
				sp := startSpinner(fmt.Sprintf("%s %s", t("root.correction_spin"), resp.Cmd))
				correctedResp, corrErr := llm.CallLLMForCorrection(
					provider, resp.Cmd, result.Stderr, context, userQuery, lang, fastMode)
				sp.stop("")

				if corrErr != nil {
					fmt.Fprintf(os.Stderr, t("root.llm_fail")+"\n", corrErr)
					fmt.Fprintln(os.Stderr, t("root.llm_nocmd"))
					os.Exit(1)
				}
				// 极速模式下，大模型返回的 explain 是空的，给它一个默认的占位提示
				if fastMode && correctedResp.Explain == "" {
					if lang == "en" {
						correctedResp.Explain = "(No explanation in fast mode)"
					} else {
						correctedResp.Explain = "(极速模式下无错误分析解释)"
					}
				}
				resp = correctedResp
				// 继续循环，展示修正后的命令
			}
		}
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
	// 注册 -f / --fast 参数
	rootCmd.Flags().BoolVarP(&fastMode, "fast", "f", false, "极速模式：跳过联网搜索和原理解释，极速生成命令")
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
	viper.SetDefault("provider", "deepseek")

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

// initialCall 首次 LLM 调用，包含搜索慢路径逻辑
func initialCall(provider llm.Provider, context string, userQuery string, lang string, fast bool) (*llm.LLMResponse, error) {
	sp := startSpinner(fmt.Sprintf("%s %s", t("root.llm_calling"), userQuery))
	resp, err := llm.CallLLM(provider, context, userQuery, lang, fast)
	if err != nil {
		sp.stop("")
		return nil, err
	}

	// 保存原始响应作为回退
	fallbackResp := resp

	// 极速模式下，直接忽略搜索逻辑，强制返回当前结果
	if fast {
		sp.stop("")
		// 如果 explain 为空，给一个默认提示
		if resp.Explain == "" {
			resp.Explain = "(极速模式下无解释内容)"
		}
		return resp, nil
	}

	// 慢路径：LLM 表示需要联网搜索
	if resp.NeedSearch {
		sp.update(t("root.search_spin"))

		searchResults, searchErr := search.Search(userQuery, 3)
		if searchErr != nil {
			sp.stop("")
			fmt.Fprintf(os.Stderr, t("root.search_fail")+"\n", searchErr)
			if resp.Cmd == "" {
				fmt.Println(t("root.search_fallback"))
			} else {
				fmt.Fprintln(os.Stderr, t("root.search_fallback"))
			}
		}

		if searchResults != nil && len(searchResults) > 0 {
			sp.update(t("root.search_spin_done"))
			flatResults := search.FlattenResults(searchResults, lang)
			resp2, err2 := llm.CallLLMWithSearch(provider, context, userQuery, flatResults, lang)
			if err2 != nil {
				sp.stop("")
				return nil, err2
			}

			if resp2.NeedSearch {
				sp.stop("")
				fmt.Println(t("root.search_still"))
			}
			resp = resp2
		} else {
			resp = fallbackResp
		}
	}

	sp.stop("")
	return resp, nil
}
