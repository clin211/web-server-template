package core

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// OnInitialize 设置需要读取的配置文件名、环境变量，并将其内容读取到 viper 中.
func OnInitialize(configFile *string, envPrefix string, loadDirs []string, defaultConfigName string) func() {
	return func() {
		// Viper 是进程级状态。清理上一次命令执行残留的路径和键，避免测试或
		// 嵌入式调用在多次构造命令时读取到旧配置。
		viper.Reset()
		if configFile != nil {
			// 从命令行选项指定的配置文件中读取
			if strings.TrimSpace(*configFile) != "" {
				viper.SetConfigFile(*configFile)
			} else {
				for _, dir := range loadDirs {
					viper.AddConfigPath(dir)
				}
				viper.SetConfigType("yaml")
				viper.SetConfigName(defaultConfigName)
			}
		} else {
			for _, dir := range loadDirs {
				// 将 dir 目录加入到配置文件的搜索路径
				viper.AddConfigPath(dir)
			}

			// 设置配置文件格式为 YAML
			viper.SetConfigType("yaml")

			// 配置文件名称（没有文件扩展名）
			viper.SetConfigName(defaultConfigName)
		}

		// 读取匹配的环境变量
		viper.AutomaticEnv()

		// 设置环境变量前缀（如 "APP"）
		viper.SetEnvPrefix(envPrefix)

		// 将 key 字符串中 '.' 和 '-' 替换为 '_'
		replacer := strings.NewReplacer(".", "_", "-", "_")
		viper.SetEnvKeyReplacer(replacer)

		// 读取配置文件。如果指定了配置文件名，则使用指定的配置文件，否则在注册的搜索路径中搜索。
		// 显式指定的文件必须存在；默认搜索路径没有命中时保留默认值，让上层
		// Validate 报告真正缺失的必填配置（例如 JWT secret）。
		if err := viper.ReadInConfig(); err != nil {
			if configFile != nil && strings.TrimSpace(*configFile) != "" {
				viper.Set("__config_error", fmt.Errorf("read config file %q: %w", *configFile, err))
			}
		}
	}
}

// ConfigError 返回 OnInitialize 记录的显式配置文件错误。
func ConfigError() error {
	value := viper.Get("__config_error")
	if err, ok := value.(error); ok {
		return err
	}
	return nil
}
