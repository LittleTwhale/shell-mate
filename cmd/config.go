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
	apiKey       string // -k, API 密钥
	modelName    string // -m, 模型名称
	apiBaseURL   string // -b, API 端点地址
	searchAPIKey string // -s, 搜索 API 密钥（预留）
	language     string // -l, 界面语言 (zh/en)
	addDanger    string // 添加高危关键词
	removeDanger string // 移除高危关键词
)

// configCmd 管理 shell-mate 的所有配置项，并持久化到 ~/.shell-mate.yaml
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "配置 shell-mate 参数",
	Long: `设置 API_KEY、API_BASE_URL、MODEL_NAME、SEARCH_API_KEY、LANGUAGE 等配置项，
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
		if language != "" {
			viper.Set("language", language)
			changed = true
		}
		// 处理添加自定义高危关键词
		if addDanger != "" {
			list := viper.GetStringSlice("dangerous_keywords")
			exists := false
			for _, v := range list {
				if v == addDanger {
					exists = true
					break
				}
			}
			if !exists {
				list = append(list, addDanger)
				viper.Set("dangerous_keywords", list)
				changed = true
			}
		}

		// 处理移除自定义高危关键词
		if removeDanger != "" {
			list := viper.GetStringSlice("dangerous_keywords")
			newList := []string{}
			for _, v := range list {
				if v != removeDanger {
					newList = append(newList, v)
				}
			}
			viper.Set("dangerous_keywords", newList)
			changed = true
		}
		if changed {
			if err := viper.WriteConfig(); err != nil {
				if _, ok := err.(viper.ConfigFileNotFoundError); ok {
					// 配置文件不存在时，创建新文件
					home, _ := os.UserHomeDir()
					cfgPath := filepath.Join(home, ".shell-mate.yaml")
					if err := viper.WriteConfigAs(cfgPath); err != nil {
						fmt.Fprintf(os.Stderr, t("config.write_err")+"\n", err)
						os.Exit(1)
					}
				} else {
					fmt.Fprintf(os.Stderr, t("config.write_err")+"\n", err)
					os.Exit(1)
				}
			}
			fmt.Println(t("config.saved"))
		} else {
			printConfig()
		}
	},
}

// printConfig 打印当前所有配置项（敏感信息脱敏显示）
func printConfig() {
	fmt.Println(t("config.title"))
	fmt.Printf(t("config.api_key")+"\n", maskValue(viper.GetString("api_key")))
	fmt.Printf(t("config.api_base")+"\n", viper.GetString("api_base_url"))
	fmt.Printf(t("config.model_name")+"\n", viper.GetString("model_name"))
	fmt.Printf(t("config.search_key")+"\n", maskValue(viper.GetString("search_api_key")))
	lang := viper.GetString("language")
	if lang == "" {
		lang = "zh (默认)"
	}
	fmt.Printf(t("config.language")+"\n", lang)

	// 打印高危关键词配置
	dangerList := viper.GetStringSlice("dangerous_keywords")
	if len(dangerList) == 0 {
		fmt.Printf(t("config.danger_list")+"\n", "(使用系统默认)")
	} else {
		displayStr := fmt.Sprintf("系统默认 + %v", dangerList)
		if getCurrentLang() == LangEN {
			displayStr = fmt.Sprintf("System Defaults + %v", dangerList)
		}
		fmt.Printf(t("config.danger_list")+"\n", displayStr)
	}
}

// maskValue 对敏感值进行脱敏处理，仅显示首尾各 4 个字符
func maskValue(v string) string {
	if v == "" {
		return t("config.unset")
	}
	if len(v) <= 8 {
		return strings.Repeat("*", len(v))
	}
	return v[:4] + strings.Repeat("*", len(v)-8) + v[len(v)-4:]
}

func init() {
	configCmd.Flags().StringVarP(&apiKey, "api-key", "k", "", "设置 LLM API Key")
	configCmd.Flags().StringVarP(&apiBaseURL, "api-base-url", "b", "", "设置 API 端点地址 (默认: https://api.deepseek.com)")
	configCmd.Flags().StringVarP(&modelName, "model-name", "m", "", "设置模型名称 (默认: deepseek-v4-flash)")
	configCmd.Flags().StringVarP(&searchAPIKey, "search-api-key", "s", "", "设置搜索 API Key (Tavily/Serper)")
	configCmd.Flags().StringVarP(&language, "language", "l", "", "设置界面语言: zh (中文) 或 en (英文)")
	configCmd.Flags().StringVar(&addDanger, "add-danger", "", "添加自定义高危命令关键词")
	configCmd.Flags().StringVar(&removeDanger, "remove-danger", "", "移除自定义高危命令关键词")
	rootCmd.AddCommand(configCmd)
}
