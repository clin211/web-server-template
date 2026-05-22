package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/clin211/gin-enterprise-template/pkg/core"
	"github.com/clin211/gin-enterprise-template/pkg/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	genericapiserver "k8s.io/apiserver/pkg/server"

	"github.com/clin211/gin-enterprise-template/cmd/gin-enterprise-template-apiserver/app/options"
)

// defaultHomeDir 与 defaultConfigName 在程序启动时基于 os.Args[0] 动态推导，
// 而不是硬编码字符串。这样让本模板被 fork / 通过 forge 重命名后无需手动改源码。
//
// 例如：
//
//	$ ls ~/.demo-api-apiserver/
//	demo-api-apiserver.yaml
//
// 任何下游使用者编译出新的二进制名（如 demo-api-apiserver）后，配置目录与
// 文件名会自动跟随该名字变化，避免「为什么找不到默认配置」这类常见困惑。
var (
	defaultHomeDir    = computeDefaultHomeDir()
	defaultConfigName = computeDefaultConfigName()
)

// 配置文件的路径
var configFile string

// computeBinaryName 提取当前可执行文件名（去除路径与 .exe 后缀），
// 并在异常情况下退化为 "apiserver" 兜底。
func computeBinaryName() string {
	binary := filepath.Base(os.Args[0])
	binary = strings.TrimSuffix(binary, ".exe")
	if binary == "" || binary == "." || binary == "/" {
		return "apiserver"
	}
	return binary
}

func computeDefaultHomeDir() string {
	return "." + computeBinaryName()
}

func computeDefaultConfigName() string {
	return computeBinaryName() + ".yaml"
}

// NewWebServerCommand 创建用于启动应用程序的 *cobra.Command 对象。
func NewWebServerCommand() *cobra.Command {
	// 创建默认的应用程序命令行选项
	opts := options.NewServerOptions()

	cmd := &cobra.Command{
		// 指定命令名称，将出现在帮助信息中
		Use: "gin-enterprise-template-apiserver",
		// 命令的简短描述
		Short: "Please update the short description of the binary file.",
		// 命令的详细描述
		Long: `Please update the detailed description of the binary file.`,
		// 当命令遇到错误时不打印帮助信息。
		// 将此设置为 true 可确保错误立即可见。
		SilenceUsage: true,
		// 指定调用 cmd.Execute() 时要执行的 Run 函数
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := genericapiserver.SetupSignalContext()

			// 如果传递了 --version 标志，则打印版本信息并退出
			version.PrintAndExitIfRequested()

			// 将 viper 的配置反序列化到 opts
			if err := viper.Unmarshal(opts); err != nil {
				return fmt.Errorf("failed to unmarshal configuration: %w", err)
			}

			// 验证命令行选项
			if err := opts.Validate(); err != nil {
				return fmt.Errorf("invalid options: %w", err)
			}
			if err := opts.OTelOptions.Apply(); err != nil {
				return err
			}
			defer func() {
				_ = opts.OTelOptions.Shutdown(ctx)
			}()

			return run(ctx, opts)
		},
		// 为命令设置参数验证。不需要命令行参数。
		// 例如：./gin-enterprise-template-apiserver param1 param2
		Args: cobra.NoArgs,
	}

	// 初始化配置函数，在每个命令运行时调用。
	// envPrefix 必须只含 [A-Z0-9_]，否则 shell 无法设置该前缀的环境变量。
	// 配合 viper.AutomaticEnv()，所有 yaml 配置项都可被环境变量覆盖：
	//   APP_JWT_SECRET            → jwt.secret
	//   APP_POSTGRESQL_PASSWORD   → postgresql.password
	//   APP_REDIS_PASSWORD        → redis.password
	cobra.OnInitialize(core.OnInitialize(&configFile, "APP", searchDirs(), defaultConfigName))

	// cobra 支持持久标志，适用于指定命令及其所有子命令。
	// 建议使用配置文件进行应用程序配置，以便更轻松地管理配置项。
	cmd.PersistentFlags().StringVarP(&configFile, "config", "c", filePath(), "Path to the gin-enterprise-template-apiserver configuration file.")

	// 将服务器选项添加为标志
	opts.AddFlags(cmd.PersistentFlags())

	// 添加 --version 标志
	version.AddFlags(cmd.PersistentFlags())

	return cmd
}

// run 包含初始化和运行服务器的主要逻辑。
func run(ctx context.Context, opts *options.ServerOptions) error {
	// 获取应用程序配置
	// 分离命令行选项和应用程序配置可以更灵活地处理这两种类型的配置。
	cfg, err := opts.Config()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// 创建并启动服务器
	server, err := cfg.NewServer(ctx)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// 运行服务器
	return server.Run(ctx)
}

// searchDirs 返回搜索配置文件的默认目录，优先级从高到低依次为：
//  1. ~/.${binary}             —— 用户私有配置（部署/产线常用位置）
//  2. ./configs                —— 仓库内置配置（默认开发体验，无需 -c 参数）
//  3. .                        —— 当前工作目录（兜底，便于自定义脚本）
//
// 由于 viper 会按顺序在这些目录里查找 ${binary}.yaml，确保仓库克隆后
// 直接 `make run` 也能命中 `configs/${binary}.yaml`。
func searchDirs() []string {
	// 获取用户的主目录。
	homeDir, err := os.UserHomeDir()
	// 如果无法获取用户的主目录，打印错误消息并退出程序。
	cobra.CheckErr(err)
	return []string{
		filepath.Join(homeDir, defaultHomeDir),
		"./configs",
		".",
	}
}

// filePath 检索默认配置文件的完整路径。
func filePath() string {
	home, err := os.UserHomeDir()
	// 如果无法检索用户的主目录，记录错误并返回空路径。
	cobra.CheckErr(err)
	return filepath.Join(home, defaultHomeDir, defaultConfigName)
}
