package log

import (
	"github.com/spf13/pflag"
	"go.uber.org/zap/zapcore"
)

// Options 包含日志的配置选项。
type Options struct {
	// DisableCaller 指定是否在日志中包含调用者信息。
	DisableCaller bool `json:"disable-caller,omitempty" mapstructure:"disable-caller"`
	// DisableStacktrace 指定是否为 panic 级别及以上的所有消息记录堆栈跟踪。
	DisableStacktrace bool `json:"disable-stacktrace,omitempty" mapstructure:"disable-stacktrace"`
	// EnableColor 指定是否输出彩色日志。
	EnableColor bool `json:"enable-color"       mapstructure:"enable-color"`
	// Level 指定最小日志级别。有效值为：debug、info、warn、error、dpanic、panic 和 fatal。
	Level string `json:"level,omitempty" mapstructure:"level"`
	// Format 指定日志输出格式。有效值为：console 和 json。
	Format string `json:"format,omitempty" mapstructure:"format"`
	// OutputPaths 指定日志的输出路径。
	OutputPaths []string `json:"output-paths,omitempty" mapstructure:"output-paths"`
}

// NewOptions 创建一个带有默认值的新 Options 对象。
func NewOptions() *Options {
	return &Options{
		Level:       zapcore.InfoLevel.String(),
		Format:      "console",
		OutputPaths: []string{"stdout"},
	}
}

// Validate 验证传递给 LogsOptions 的标志。
func (o *Options) Validate() []error {
	errs := []error{}

	return errs
}

// AddFlags 为配置添加命令行标志。
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Level, "log.level", o.Level, "最小日志输出 `LEVEL`。")
	fs.BoolVar(&o.DisableCaller, "log.disable-caller", o.DisableCaller, "禁用日志中调用者信息的输出。")
	fs.BoolVar(&o.DisableStacktrace, "log.disable-stacktrace", o.DisableStacktrace, ""+
		"禁用日志为 panic 级别及以上的所有消息记录堆栈跟踪。")
	fs.BoolVar(&o.EnableColor, "log.enable-color", o.EnableColor, "在纯文本格式日志中启用 ANSI 颜色输出。")
	fs.StringVar(&o.Format, "log.format", o.Format, "日志输出 `FORMAT`，支持 plain 或 json 格式。")
	fs.StringSliceVar(&o.OutputPaths, "log.output-paths", o.OutputPaths, "日志的输出路径。")
}
