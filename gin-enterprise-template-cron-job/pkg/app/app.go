package app

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	_ "go.uber.org/automaxprocs"
	"k8s.io/component-base/cli"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/term"

	"github.com/clin211/gin-enterprise-template/pkg/log"
	genericoptions "github.com/clin211/gin-enterprise-template/pkg/options"
	"github.com/clin211/gin-enterprise-template/pkg/version"
)

// App 是 CLI 应用程序的主结构。
// 建议使用 app.NewApp() 函数创建应用。
type App struct {
	name        string
	shortDesc   string
	description string
	run         RunFunc
	cmd         *cobra.Command
	args        cobra.PositionalArgs

	// +optional
	healthCheckFunc HealthCheckFunc

	// +optional
	options any

	// +optional
	silence bool

	// +optional
	noConfig bool

	// 监视和重新读取配置文件
	// +optional
	watch bool

	contextExtractors map[string]func(context.Context) string
}

// RunFunc 定义应用程序的启动回调函数。
type RunFunc func() error

// HealthCheckFunc 定义应用程序的健康检查函数。
type HealthCheckFunc func() error

// Option 定义用于初始化应用程序
// 结构的可选参数。
type Option func(*App)

// WithOptions 用于打开应用程序从命令行读取
// 或从配置文件读取参数的功能。
func WithOptions(opts any) Option {
	return func(app *App) {
		app.options = opts
	}
}

// WithRunFunc 用于设置应用程序启动回调函数选项。
func WithRunFunc(run RunFunc) Option {
	return func(app *App) {
		app.run = run
	}
}

// WithDescription 用于设置应用程序的描述。
func WithDescription(desc string) Option {
	return func(app *App) {
		app.description = desc
	}
}

// WithHealthCheckFunc 用于为应用程序设置健康检查函数。
// 应用程序框架将使用该函数启动健康检查服务器。
func WithHealthCheckFunc(fn HealthCheckFunc) Option {
	return func(app *App) {
		app.healthCheckFunc = fn
	}
}

// WithDefaultHealthCheckFunc 设置默认健康检查函数。
func WithDefaultHealthCheckFunc() Option {
	fn := func() HealthCheckFunc {
		return func() error {
			go genericoptions.NewHealthOptions().ServeHealthCheck()

			return nil
		}
	}

	return WithHealthCheckFunc(fn())
}

// WithSilence 将应用程序设置为静默模式，在该模式下，程序启动
// 信息、配置信息和版本信息不会
// 打印在控制台中。
func WithSilence() Option {
	return func(app *App) {
		app.silence = true
	}
}

// WithNoConfig 设置应用程序不提供配置标志。
func WithNoConfig() Option {
	return func(app *App) {
		app.noConfig = true
	}
}

// WithValidArgs 设置验证函数以验证非标志参数。
func WithValidArgs(args cobra.PositionalArgs) Option {
	return func(app *App) {
		app.args = args
	}
}

// WithDefaultValidArgs 设置默认验证函数以验证非标志参数。
func WithDefaultValidArgs() Option {
	return func(app *App) {
		app.args = cobra.NoArgs
	}
}

// WithWatchConfig 监视并重新读取配置文件。
func WithWatchConfig() Option {
	return func(app *App) {
		app.watch = true
	}
}

func WithLoggerContextExtractor(contextExtractors map[string]func(context.Context) string) Option {
	return func(app *App) {
		app.contextExtractors = contextExtractors
	}
}

// NewApp 根据给定的应用程序名称、
// 二进制名称和其他选项创建新的应用程序实例。
func NewApp(name string, shortDesc string, opts ...Option) *App {
	app := &App{
		name:      name,
		run:       func() error { return nil },
		shortDesc: shortDesc,
	}

	for _, o := range opts {
		o(app)
	}

	app.buildCommand()

	return app
}

// buildCommand 用于构建 cobra 命令。
func (app *App) buildCommand() {
	cmd := &cobra.Command{
		Use:   formatBaseName(app.name),
		Short: app.shortDesc,
		Long:  app.description,
		RunE:  app.runCommand,
		PersistentPreRunE: func(*cobra.Command, []string) error {
			return nil
		},
		Args: app.args,
	}
	// 当为 Cobra 命令启用错误打印时，标志解析
	// 错误首先被打印，然后可选地打印通常很长的用法
	// 文本。这在控制台中非常不可读，因为屏幕上可见的
	// 最后几行不包含错误。
	//
	// #sig-cli 的建议是打印用法文本，然后
	// 打印错误。我们在这里为所有命令一致地实现这一点。
	// 但是，我们不想在命令因解析以外的原因
	// 执行失败时打印用法文本。我们通过
	// FlagParseError 回调来检测这一点。
	//
	// 某些命令（如 kubectl）已经自己处理了这个问题。
	// 我们不更改这些命令的行为。
	if !cmd.SilenceUsage {
		cmd.SilenceUsage = true
		cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
			// 重新启用用法打印。
			c.SilenceUsage = false
			return err
		})
	}
	// 在所有情况下，错误打印都在下面完成。
	cmd.SilenceErrors = true

	cmd.SetOutput(os.Stdout)
	cmd.SetErr(os.Stderr)
	cmd.Flags().SortFlags = true

	var fs *pflag.FlagSet
	// 方法2：使用type switch
	switch typed := app.options.(type) {
	case NamedFlagSetOptions:
		var fss cliflag.NamedFlagSets
		fs = fss.FlagSet("global")

		if app.options != nil {
			fss = typed.Flags()
		}

		for _, f := range fss.FlagSets {
			cmd.Flags().AddFlagSet(f)
		}

		cols, _, _ := term.TerminalSize(cmd.OutOrStdout())
		cliflag.SetUsageAndHelpFunc(cmd, fss, cols)
	case FlagSetOptions:
		fs = cmd.PersistentFlags()
		if app.options != nil {
			typed.AddFlags(fs)
		}
	default:
	}

	version.AddFlags(fs)

	if !app.noConfig {
		AddConfigFlag(fs, app.name, app.watch)
	}

	app.cmd = cmd
}

// Run 用于启动应用程序。
func (app *App) Run() {
	os.Exit(cli.Run(app.cmd))
}

func (app *App) runCommand(cmd *cobra.Command, args []string) error {
	// 显示应用程序版本信息
	version.PrintAndExitIfRequested()

	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		return err
	}

	if app.options != nil {
		if err := viper.Unmarshal(app.options); err != nil {
			return err
		}

		if complete, ok := app.options.(interface{ Complete() error }); ok {
			if err := complete.Complete(); err != nil {
				return err
			}
		}

		if validate, ok := app.options.(interface{ Validate() error }); ok {
			if err := validate.Validate(); err != nil {
				return err
			}
		}
	}

	app.initializeLogger()

	if !app.silence {
		slog.Info("Starting application",
			"name", app.name,
			"version", version.Get().ToJSON())
		slog.Info("Golang settings",
			"GOGC", os.Getenv("GOGC"),
			"GOMAXPROCS", os.Getenv("GOMAXPROCS"),
			"GOTRACEBACK", os.Getenv("GOTRACEBACK"))
		if !app.noConfig {
			PrintConfig()
		} else if app.options != nil {
			cliflag.PrintFlags(cmd.Flags())
		}
	}

	if app.healthCheckFunc != nil {
		if err := app.healthCheckFunc(); err != nil {
			return err
		}
	}

	// 运行应用程序
	return app.run()
}

// Command 返回应用程序内的 cobra 命令实例。
func (app *App) Command() *cobra.Command {
	return app.cmd
}

// formatBaseName 根据给定的名称将
// 其格式化为不同操作系统下的可执行文件名。
func formatBaseName(name string) string {
	// 不区分大小写并删除可执行文件后缀（如果存在）
	if runtime.GOOS == "windows" {
		name = strings.ToLower(name)
		name = strings.TrimSuffix(name, ".exe")
	}
	return name
}

// initializeLogger 根据配置设置日志系统。
func (app *App) initializeLogger() {
	logOptions := log.NewOptions()

	// 从 viper 配置日志选项
	if viper.IsSet("log.disable-caller") {
		logOptions.DisableCaller = viper.GetBool("log.disable-caller")
	}
	if viper.IsSet("log.disable-stacktrace") {
		logOptions.DisableStacktrace = viper.GetBool("log.disable-stacktrace")
	}
	if viper.IsSet("log.level") {
		logOptions.Level = viper.GetString("log.level")
	}
	if viper.IsSet("log.format") {
		logOptions.Format = viper.GetString("log.format")
	}
	if viper.IsSet("log.output-paths") {
		logOptions.OutputPaths = viper.GetStringSlice("log.output-paths")
	}

	// 使用自定义上下文提取器初始化日志
	log.Init(logOptions, log.WithContextExtractor(app.contextExtractors))
}
