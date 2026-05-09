package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// config 命令的命令行参数
var (
	apiKey      string // -k, API 密钥
	modelName   string // -m, 模型名称
	apiBaseURL  string // -b, API 端点地址
	searchAPIKey string // -s, 搜索 API 密钥（预留）
)

// configCmd 管理 shell-mate 的所有配置项，并持久化到 ~/.shell-mate.yaml
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "配置 shell-mate 参数",
	Long: `设置 API_KEY、API_BASE_URL、MODEL_NAME、SEARCH_API_KEY 等配置项，
并将它们持久化保存到 ~/.shell-mate.yaml 文件中。

不带任何参数运行时，显示当前配置。`,
	Run: func(cmd *cobra.Command, args []string) {
		changed := false

		if apiKey != "" {
			viper.Set("api_key", apiKey)
			changed = true
		}
		if apiBaseURL != "" {
			viper.Set("api_base_url", apiBaseURL)
			changed = true
		}
		if modelName != "" {
			viper.Set("model_name", modelName)
			changed = true
		}
		if searchAPIKey != "" {
			viper.Set("search_api_key", searchAPIKey)
			changed = true
		}

		if changed {
			if err := viper.WriteConfig(); err != nil {
				if _, ok := err.(viper.ConfigFileNotFoundError); ok {
					// 配置文件不存在时，创建新文件
					home, _ := os.UserHomeDir()
					cfgPath := filepath.Join(home, ".shell-mate.yaml")
					if err := viper.WriteConfigAs(cfgPath); err != nil {
						fmt.Fprintf(os.Stderr, "写入配置文件失败: %s\n", err)
						os.Exit(1)
					}
				} else {
					fmt.Fprintf(os.Stderr, "写入配置文件失败: %s\n", err)
					os.Exit(1)
				}
			}
			fmt.Println("配置已保存到 ~/.shell-mate.yaml")
		} else {
			printConfig()
		}
	},
}

// printConfig 打印当前所有配置项（敏感信息脱敏显示）
func printConfig() {
	fmt.Println("当前配置 (~/.shell-mate.yaml):")
	fmt.Printf("  API_KEY        : %s\n", maskValue(viper.GetString("api_key")))
	fmt.Printf("  API_BASE_URL   : %s\n", viper.GetString("api_base_url"))
	fmt.Printf("  MODEL_NAME     : %s\n", viper.GetString("model_name"))
	fmt.Printf("  SEARCH_API_KEY : %s\n", maskValue(viper.GetString("search_api_key")))
}

// maskValue 对敏感值进行脱敏处理，仅显示首尾各 4 个字符
func maskValue(v string) string {
	if v == "" {
		return "(未设置)"
	}
	if len(v) <= 8 {
		return strings.Repeat("*", len(v))
	}
	return v[:4] + strings.Repeat("*", len(v)-8) + v[len(v)-4:]
}

func init() {
	configCmd.Flags().StringVarP(&apiKey, "api-key", "k", "", "设置 LLM API Key")
	configCmd.Flags().StringVarP(&apiBaseURL, "api-base-url", "b", "", "设置 API 端点地址 (默认: https://api.openai.com/v1)")
	configCmd.Flags().StringVarP(&modelName, "model-name", "m", "", "设置模型名称 (默认: gpt-4o-mini)")
	configCmd.Flags().StringVarP(&searchAPIKey, "search-api-key", "s", "", "设置搜索 API Key (Tavily/Serper)")

	rootCmd.AddCommand(configCmd)
}
