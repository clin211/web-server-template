package app

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"k8s.io/client-go/util/homedir"
)

const configFlagName = "config"

var cfgFile string

// AddConfigFlag 为特定服务器将标志添加到指定的 FlagSet 对象。
// 它还设置传递的函数，以便在调用每个 cobra 命令的 Execute 方法时
// 将配置文件中的值读取到 viper 中。
func AddConfigFlag(fs *pflag.FlagSet, name string, watch bool) {
	fs.AddFlag(pflag.Lookup(configFlagName))

	// 启用 viper 的自动环境变量解析。这意味着
	// viper 将自动从环境变量中读取与 viper
	// 变量对应的值。
	viper.AutomaticEnv()
	// 设置环境变量前缀。使用 strings.ReplaceAll 函数
	// 将名称中的连字符替换为下划线，并使用 strings.ToUpper
	// 将名称转换为大写，然后将其设置为环境变量的前缀。
	viper.SetEnvPrefix(strings.ReplaceAll(strings.ToUpper(name), "-", "_"))
	// 设置环境变量键的替换规则。使用
	// strings.NewReplacer 函数指定将句点和连字符替换为下划线。
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	cobra.OnInitialize(func() {
		if cfgFile != "" {
			viper.SetConfigFile(cfgFile)
		} else {
			viper.AddConfigPath(".")

			if names := strings.Split(name, "-"); len(names) > 1 {
				viper.AddConfigPath(filepath.Join(homedir.HomeDir(), "."+names[0]))
				viper.AddConfigPath(filepath.Join("/etc", names[0]))
			}

			viper.SetConfigName(name)
		}

		if err := viper.ReadInConfig(); err != nil {
			slog.LogAttrs(nil, slog.LevelDebug, "读取配置文件失败",
				slog.String("file", cfgFile),
				slog.Any("err", err))
		} else {
			slog.LogAttrs(nil, slog.LevelDebug, "成功读取配置文件",
				slog.String("file", viper.ConfigFileUsed()))
		}

		if watch {
			viper.WatchConfig()
			viper.OnConfigChange(func(e fsnotify.Event) {
				slog.LogAttrs(nil, slog.LevelInfo, "配置文件已更改",
					slog.String("name", e.Name))
			})
		}
	})
}

func PrintConfig() {
	for _, key := range viper.AllKeys() {
		slog.LogAttrs(nil, slog.LevelDebug, fmt.Sprintf("CFG: %s=%v", key, viper.Get(key)))
	}
}

func init() {
	pflag.StringVarP(&cfgFile, configFlagName, "c", cfgFile, "从指定的 `FILE` 读取配置，"+
		"支持 JSON、TOML、YAML、HCL 或 Java properties 格式。")
}
