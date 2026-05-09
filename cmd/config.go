package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	openaiAPIKey string
	modelName    string
	searchAPIKey string
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "配置 shell-mate 参数",
	Long: `设置 OPENAI_API_KEY、MODEL_NAME、SEARCH_API_KEY 等配置项，
并将它们持久化保存到 ~/.shell-mate.yaml 文件中。

不带任何参数运行时，显示当前配置。`,
	Run: func(cmd *cobra.Command, args []string) {
		changed := false

		if openaiAPIKey != "" {
			viper.Set("openai_api_key", openaiAPIKey)
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

func printConfig() {
	fmt.Println("当前配置 (~/.shell-mate.yaml):")
	fmt.Printf("  OPENAI_API_KEY  : %s\n", maskValue(viper.GetString("openai_api_key")))
	fmt.Printf("  MODEL_NAME      : %s\n", viper.GetString("model_name"))
	fmt.Printf("  SEARCH_API_KEY  : %s\n", maskValue(viper.GetString("search_api_key")))
}

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
	configCmd.Flags().StringVarP(&openaiAPIKey, "openai-api-key", "k", "", "设置 OpenAI API Key")
	configCmd.Flags().StringVarP(&modelName, "model-name", "m", "", "设置模型名称 (默认: gpt-4o-mini)")
	configCmd.Flags().StringVarP(&searchAPIKey, "search-api-key", "s", "", "设置搜索 API Key (Tavily/Serper)")

	rootCmd.AddCommand(configCmd)
}
