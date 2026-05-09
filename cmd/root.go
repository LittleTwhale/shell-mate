package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"shell-mate/llm"
)

var cfgFile string

// rootCmd 是 shell-mate 的根命令，接收自然语言描述并翻译为 Shell 命令
var rootCmd = &cobra.Command{
	Use:   "sm [自然语言描述]",
	Short: "shell-mate - 终端命令行 AI 助手",
	Long: `shell-mate 是一个基于 Go 开发的终端命令行 AI 助手，
能将你的自然语言请求直接翻译成可执行的 Shell 命令。`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cobraCmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cobraCmd.Help()
			return
		}

		// 读取 API 密钥，兼容旧的 openai_api_key 配置项
		apiKey := viper.GetString("api_key")
		if apiKey == "" {
			apiKey = viper.GetString("openai_api_key")
		}
		if apiKey == "" {
			fmt.Fprintln(os.Stderr, "错误: 请先设置 API_KEY，运行: sm config -k <your-api-key>")
			os.Exit(1)
		}

		// 读取 API 端点地址和模型名称
		apiBaseURL := viper.GetString("api_base_url")
		modelName := viper.GetString("model_name")

		// 收集系统上下文并调用 LLM
		context := llm.GatherContext()
		fmt.Printf("正在请求 AI 翻译: %s\n\n", args[0])

		resp, err := llm.CallLLM(apiBaseURL, apiKey, modelName, context, args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "调用 AI 失败: %v\n", err)
			os.Exit(1)
		}

		// 输出翻译结果
		fmt.Printf("命令: %s\n", resp.Cmd)
		fmt.Printf("解释: %s\n", resp.Explain)
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
			fmt.Fprintln(os.Stderr, "无法获取用户主目录:", err)
			os.Exit(1)
		}
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".shell-mate")
	}

	// 设置默认值
	viper.SetDefault("api_base_url", "https://api.openai.com/v1")
	viper.SetDefault("model_name", "gpt-4o-mini")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, "读取配置文件出错: %s\n", err)
		}
	}
}
