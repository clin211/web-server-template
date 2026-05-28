package options

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
)

var _ IOptions = (*SlogOptions)(nil)

// SlogOptions 包含与 slog 相关的配置项。
type SlogOptions struct {
	// Level 指定要输出的最低日志级别。
	// 可能的值：debug、info、warn、error
	Level string `json:"level,omitempty" mapstructure:"level"`
	// AddSource 将源代码位置（文件:行）添加到日志记录
	AddSource bool `json:"add-source,omitempty" mapstructure:"add-source"`
	// Format 指定日志消息的结构。
	// 可能的值：json、text
	Format string `json:"format,omitempty" mapstructure:"format"`
	// TimeFormat 指定文本输出的时间格式。
	// 使用 Go 时间格式布局。空值表示 RFC3339。
	TimeFormat string `json:"time-format,omitempty" mapstructure:"time-format"`
	// Output 指定写入日志的位置。
	// 可能的值：stdout、stderr 或文件路径
	Output string `json:"output,omitempty" mapstructure:"output"`
}

// NewSlogOptions 创建带有默认参数的 Options 对象。
func NewSlogOptions() *SlogOptions {
	return &SlogOptions{
		Level:      "info",
		AddSource:  false,
		Format:     "text",
		TimeFormat: "",
		Output:     "stdout",
	}
}

// Validate 验证传递给 SlogOptions 的标志。
func (o *SlogOptions) Validate() []error {
	var errs []error

	// 验证日志级别
	switch strings.ToUpper(strings.TrimSpace(o.Level)) {
	case "DEBUG", "INFO", "WARN", "WARNING", "ERROR":
	default:
		errs = append(errs, fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", o.Level))
	}

	// 验证格式
	switch o.Format {
	case "json", "text":
	default:
		errs = append(errs, fmt.Errorf("invalid log format: %s (must be json or text)", o.Format))
	}

	// 验证输出
	if o.Output != "stdout" && o.Output != "stderr" && o.Output != "" {
		// 检查是否为有效的文件路径（基本验证）
		if !filepath.IsAbs(o.Output) && !strings.Contains(o.Output, "/") {
			errs = append(errs, fmt.Errorf("invalid output path: %s", o.Output))
		}
	}

	return errs
}

// AddFlags 为配置添加命令行标志。
func (o *SlogOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringVar(&o.Level, fullPrefix+".level", o.Level, "Sets the log level. Permitted levels: debug, info, warn, error.")
	fs.StringVar(&o.Format, fullPrefix+".format", o.Format, "Sets the log format. Permitted formats: json, text.")
	fs.BoolVar(&o.AddSource, fullPrefix+".add-source", o.AddSource, "Add source file:line to log records.")
	fs.StringVar(&o.TimeFormat, fullPrefix+".time-format", o.TimeFormat, ""+
		"Time format for text logs using Go's time layout format. Leave empty for RFC3339. "+
		"Examples: '2006-01-02 15:04:05'")
	fs.StringVar(&o.Output, fullPrefix+".output", o.Output, "Log output destination (stdout, stderr, or file path).")
}

// ToSlogLevel 将字符串级别转换为 slog.Level。
func (o *SlogOptions) ToSlogLevel() slog.Level {
	switch strings.ToUpper(strings.TrimSpace(o.Level)) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		// 如果未知级别，默认为 INFO
		return slog.LevelInfo
	}
}

// GetWriter 根据输出配置返回适当的 io.Writer。
func (o *SlogOptions) GetWriter() (io.Writer, error) {
	switch o.Output {
	case "stdout", "":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	default:
		// 文件输出
		file, err := os.OpenFile(o.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file %s: %w", o.Output, err)
		}
		return file, nil
	}
}

// BuildHandler 根据配置创建 slog.Handler。
func (o *SlogOptions) BuildHandler() (slog.Handler, error) {
	writer, err := o.GetWriter()
	if err != nil {
		return nil, err
	}

	opts := &slog.HandlerOptions{
		Level:     o.ToSlogLevel(),
		AddSource: o.AddSource,
	}

	// 为文本处理程序设置自定义时间格式
	if o.Format == "text" && o.TimeFormat != "" {
		opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.String(slog.TimeKey, a.Value.Time().Format(o.TimeFormat))
			}
			return a
		}
	}

	var handler slog.Handler
	switch o.Format {
	case "json":
		handler = slog.NewJSONHandler(writer, opts)
	case "text":
		handler = slog.NewTextHandler(writer, opts)
	default:
		handler = slog.NewTextHandler(writer, opts)
	}

	return handler, nil
}

// BuildLogger 构造并返回已配置的 slog.Logger 实例，而不影响全局日志记录器。
func (o *SlogOptions) BuildLogger() (*slog.Logger, error) {
	handler, err := o.BuildHandler()
	if err != nil {
		return nil, err
	}
	return slog.New(handler), nil
}

// Apply 将配置应用于全局默认 slog 日志记录器。
func (o *SlogOptions) Apply() error {
	logger, err := o.BuildLogger()
	if err != nil {
		return err
	}
	slog.SetDefault(logger)
	return nil
}
