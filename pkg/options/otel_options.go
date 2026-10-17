package options

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	emptyexporter "github.com/clin211/gin-enterprise-template/pkg/otel/exporter/empty"
	"github.com/spf13/pflag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/clin211/gin-enterprise-template/pkg/otelslog"
)

// 确保接口实现
var _ IOptions = (*OTelOptions)(nil)

// OutputMode 表示 OpenTelemetry 数据的输出模式
type OutputMode string

const (
	OutputModeOTLP    OutputMode = "otel"    // 发送到 OpenTelemetry Collector
	OutputModeFile    OutputMode = "file"    // 输出到文件
	OutputModeConsole OutputMode = "console" // 输出到标准输出
	OutputModeClassic OutputMode = "classic" // 使用传统指标 + 日志（无 OTel）

	// OutputModeHybrid 混合输出日志、指标和追踪：
	// log->stdout, metric->prometheus, trace->otel
	OutputModeHybrid OutputMode = "hybrid"
)

// String 实现 Stringer 接口
func (o OutputMode) String() string {
	return string(o)
}

// IsValid 验证输出模式
func (o OutputMode) IsValid() bool {
	switch o {
	case OutputModeConsole, OutputModeFile, OutputModeOTLP, OutputModeClassic, OutputModeHybrid:
		return true
	default:
		return false
	}
}

// outputModeFlag 为 OutputMode 实现 pflag.Value 接口
type outputModeFlag OutputMode

func (f *outputModeFlag) String() string { return string(*f) }
func (f *outputModeFlag) Type() string   { return "string" }
func (f *outputModeFlag) Set(s string) error {
	mode := OutputMode(s)
	if !mode.IsValid() {
		return fmt.Errorf("%w: invalid output mode: %s, valid options: otlp, console, file, classic, hybrid", errInvalidConfiguration, s)
	}
	*f = outputModeFlag(mode)
	return nil
}

// Provider 包装具有关闭功能的 OpenTelemetry 提供程序
type Provider interface {
	Shutdown(context.Context) error
}

// OTelProviders 保存所有 OpenTelemetry 提供程序
type OTelProviders struct {
	tracer *trace.TracerProvider
	meter  *metric.MeterProvider
	logger *log.LoggerProvider
}

// OTelOptions OpenTelemetry 配置
type OTelOptions struct {
	// 连接设置
	Endpoint string `json:"endpoint,omitempty" mapstructure:"endpoint"`
	Insecure bool   `json:"insecure,omitempty" mapstructure:"insecure"`

	// 服务标识
	ServiceName       string `json:"service-name,omitempty" mapstructure:"service-name"`
	ServiceVersion    string `json:"service-version,omitempty" mapstructure:"service-version"`
	ServiceInstanceID string `json:"service-instance-id,omitempty" mapstructure:"service-instance-id"`
	Environment       string `json:"environment,omitempty" mapstructure:"environment"`

	// 行为设置
	SamplingRatio float64 `json:"sampling-ratio,omitempty" mapstructure:"sampling-ratio"`
	WithResource  bool    `json:"with-resource,omitempty" mapstructure:"with-resource"`

	// 输出配置
	OutputMode OutputMode `json:"output-mode,omitempty" mapstructure:"output-mode"`
	OutputDir  string     `json:"output-dir,omitempty" mapstructure:"output-dir"`

	// 日志配置
	Level     string `json:"level,omitempty" mapstructure:"level"`
	AddSource bool   `json:"add-source,omitempty" mapstructure:"add-source"`

	Slog *SlogOptions `json:"slog,omitempty" mapstructure:"slog"`

	// 内部状态
	mu         sync.RWMutex
	resourceMu sync.Mutex
	providers  *OTelProviders
	files      []io.Closer
	resource   *resource.Resource
}

// NewOTelOptions 创建具有合理默认值的新 OTelOptions
func NewOTelOptions() *OTelOptions {
	hostname, _ := os.Hostname()
	opts := &OTelOptions{
		ServiceName:       "unknown-service",
		ServiceVersion:    "1.0.0",
		ServiceInstanceID: hostname,
		Environment:       "development",
		Endpoint:          "localhost:4317",
		Insecure:          true,
		WithResource:      false,
		SamplingRatio:     1.0,
		OutputMode:        OutputModeClassic,
		OutputDir:         "./otel-output",
		Level:             "info",
		AddSource:         false,
		Slog:              NewSlogOptions(),
		providers:         &OTelProviders{},
		files:             make([]io.Closer, 0),
	}

	// 同步 slog 选项
	opts.syncSlogOptions()
	return opts
}

// syncSlogOptions 将 slog 选项与主选项同步
func (o *OTelOptions) syncSlogOptions() {
	if o.Slog != nil {
		o.Slog.Level = o.Level
		o.Slog.AddSource = o.AddSource
	}
}

// Validate 验证配置
func (o *OTelOptions) Validate() []error {
	var errs []error

	// 验证前同步 slog 选项
	o.syncSlogOptions()

	// 验证 slog 选项
	if o.Slog != nil {
		errs = append(errs, o.Slog.Validate()...)
	}

	// 验证输出模式
	if !o.OutputMode.IsValid() {
		errs = append(errs, fmt.Errorf("%w: invalid output mode: %s", errInvalidConfiguration, o.OutputMode))
	}

	errs = append(errs, o.validateSignals()...)
	return errs
}

func (o *OTelOptions) validateSignals() []error {
	var errs []error
	// 验证必填字段
	if o.ServiceName == "" {
		errs = append(errs, fmt.Errorf("%w: service name is required", errInvalidConfiguration))
	}
	if o.ServiceInstanceID == "" {
		errs = append(errs, fmt.Errorf("%w: service instance ID is required", errInvalidConfiguration))
	}

	// 验证 OTLP 特定设置
	if o.OutputMode == OutputModeOTLP && o.Endpoint == "" {
		errs = append(errs, fmt.Errorf("%w: endpoint is required for OTLP output mode", errInvalidConfiguration))
	}

	// 验证采样率
	if o.SamplingRatio < 0 || o.SamplingRatio > 1 {
		errs = append(errs, fmt.Errorf("%w: sampling ratio must be between 0 and 1, got: %f", errInvalidConfiguration, o.SamplingRatio))
	}

	// 验证文件模式的输出目录
	if o.OutputMode == OutputModeFile && o.OutputDir == "" {
		errs = append(errs, fmt.Errorf("%w: output directory is required for file output mode", errInvalidConfiguration))
	}

	return errs
}

// AddFlags 添加命令行标志
func (o *OTelOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringVar(&o.ServiceName, fullPrefix+".service-name", o.ServiceName, "Service name")
	fs.StringVar(&o.ServiceVersion, fullPrefix+".service-version", o.ServiceVersion, "Service version")
	fs.StringVar(&o.ServiceInstanceID, fullPrefix+".service-instance-id", o.ServiceInstanceID, "Service instance ID (auto-generated if empty)")
	fs.StringVar(&o.Environment, fullPrefix+".environment", o.Environment, "Environment")
	fs.StringVar(&o.Endpoint, fullPrefix+".endpoint", o.Endpoint, "OTLP endpoint")
	fs.BoolVar(&o.Insecure, fullPrefix+".insecure", o.Insecure, "Use insecure connection")
	fs.Float64Var(&o.SamplingRatio, fullPrefix+".sampling-ratio", o.SamplingRatio, "Sampling ratio (0.0-1.0)")
	fs.BoolVar(&o.WithResource, fullPrefix+".with-resource", o.WithResource, "Include system resource information")
	fs.Var((*outputModeFlag)(&o.OutputMode), fullPrefix+".output-mode", "Output mode: otlp, console, file, classic, hybrid")
	fs.StringVar(&o.OutputDir, fullPrefix+".output-dir", o.OutputDir, "Output directory for file mode")
	fs.StringVar(&o.Level, fullPrefix+".level", o.Level, "Log level: debug, info, warn, error")
	fs.BoolVar(&o.AddSource, fullPrefix+".add-source", o.AddSource, "Add source code position to logs")
	if o.Slog != nil {
		o.Slog.AddFlags(fs, fullPrefix+".slog")
	}
}

// GetResource 创建或返回缓存的资源配置
func (o *OTelOptions) GetResource() *resource.Resource {
	return o.GetResourceContext(context.Background())
}

// GetResourceContext 使用调用方上下文检测资源，并缓存当前选项实例的资源。
func (o *OTelOptions) GetResourceContext(ctx context.Context) *resource.Resource {
	if o == nil {
		return resource.Default()
	}

	o.resourceMu.Lock()
	defer o.resourceMu.Unlock()

	if o.resource != nil {
		return o.resource
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 基本资源属性
	attrs := []resource.Option{
		resource.WithAttributes(
			semconv.ServiceName(o.ServiceName),
			semconv.ServiceVersion(o.ServiceVersion),
			semconv.ServiceInstanceID(o.ServiceInstanceID),
			semconv.DeploymentEnvironment(o.Environment),
		),
	}

	// 如果启用，添加系统资源信息
	if o.WithResource {
		attrs = append(attrs,
			resource.WithOS(),
			resource.WithProcess(),
			resource.WithContainer(),
			resource.WithHost(),
		)
	}

	res, err := resource.New(ctx, attrs...)
	if err != nil {
		res = resource.Default()
	}
	o.resource = res
	return o.resource
}

// createFileWriter 创建并管理文件写入器
func (o *OTelOptions) createFileWriter(name string) (io.Writer, error) {
	if err := os.MkdirAll(o.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	filename := filepath.Join(o.OutputDir, name+".json")
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file %s: %w", filename, err)
	}

	o.files = append(o.files, file)
	return file, nil
}

// initTraces 初始化追踪
func (o *OTelOptions) initTraces(ctx context.Context) error {
	var (
		exporter trace.SpanExporter
		err      error
	)

	switch o.OutputMode {
	case OutputModeClassic:
		exporter = emptyexporter.NewEmptyExporter()
	case OutputModeOTLP, OutputModeHybrid:
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(o.Endpoint),
		}
		if o.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		exporter, err = otlptracegrpc.New(ctx, opts...)
	case OutputModeFile:
		writer, writerErr := o.createFileWriter("traces")
		if writerErr != nil {
			return fmt.Errorf("failed to create trace writer: %w", writerErr)
		}
		exporter, err = stdouttrace.New(stdouttrace.WithWriter(writer))
	default: // OutputModeConsole and fallback
		exporter, err = stdouttrace.New(stdouttrace.WithWriter(os.Stdout))
	}

	if err != nil {
		return fmt.Errorf("failed to create trace exporter: %w", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(o.GetResourceContext(ctx)),
		trace.WithSampler(trace.TraceIDRatioBased(o.SamplingRatio)),
	)

	o.providers.tracer = tp
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return nil
}

// initMetrics 初始化指标
func (o *OTelOptions) initMetrics(ctx context.Context) error {
	var (
		reader   metric.Reader
		exporter metric.Exporter
		err      error
	)

	switch o.OutputMode {
	case OutputModeClassic, OutputModeHybrid:
		reader, err = prometheus.New()
	case OutputModeOTLP:
		opts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(o.Endpoint)}
		if o.Insecure {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}
		exporter, err = otlpmetricgrpc.New(ctx, opts...)
	case OutputModeFile:
		writer, writerErr := o.createFileWriter("metrics")
		if writerErr != nil {
			return fmt.Errorf("failed to create metrics writer: %w", writerErr)
		}
		exporter, err = stdoutmetric.New(stdoutmetric.WithWriter(writer))
	default: // OutputModeConsole and fallback
		exporter, err = stdoutmetric.New(stdoutmetric.WithWriter(os.Stdout))
	}

	if err != nil {
		return fmt.Errorf("failed to create metrics exporter: %w", err)
	}

	if exporter != nil {
		reader = metric.NewPeriodicReader(exporter)
	}

	mp := metric.NewMeterProvider(
		metric.WithReader(reader),
		metric.WithResource(o.GetResourceContext(ctx)),
	)

	o.providers.meter = mp
	otel.SetMeterProvider(mp)
	return nil
}

// initLogs 初始化日志
func (o *OTelOptions) initLogs(ctx context.Context) error {
	var (
		exporter log.Exporter
		err      error
	)

	switch o.OutputMode {
	case OutputModeClassic, OutputModeHybrid:
		return o.Slog.Apply()
	case OutputModeOTLP:
		opts := []otlploggrpc.Option{
			otlploggrpc.WithEndpoint(o.Endpoint),
		}
		if o.Insecure {
			opts = append(opts, otlploggrpc.WithInsecure())
		}
		exporter, err = otlploggrpc.New(ctx, opts...)
	case OutputModeFile:
		writer, writerErr := o.createFileWriter("logs")
		if writerErr != nil {
			return fmt.Errorf("failed to create logs writer: %w", writerErr)
		}
		exporter, err = stdoutlog.New(stdoutlog.WithWriter(writer))
	default: // OutputModeConsole and fallback
		exporter, err = stdoutlog.New(stdoutlog.WithWriter(os.Stdout))
	}

	if err != nil {
		return fmt.Errorf("failed to create log exporter: %w", err)
	}

	var processor log.Processor
	if o.OutputMode == OutputModeConsole {
		processor = log.NewSimpleProcessor(exporter)
	} else {
		processor = log.NewBatchProcessor(exporter, log.WithExportTimeout(5*time.Second))
	}

	lp := log.NewLoggerProvider(
		log.WithProcessor(processor),
		log.WithResource(o.GetResourceContext(ctx)),
	)

	o.providers.logger = lp
	otel.SetLoggerProvider(lp)

	// 设置 slog 集成
	logger := otelslog.NewLogger(
		o.ServiceName,
		otelslog.WithLoggerProvider(otel.GetLoggerProvider()),
		otelslog.WithSource(o.AddSource),
		otelslog.WithLevelString(o.Level),
	)
	slog.SetDefault(logger)

	return nil
}

// Apply 应用配置
func (o *OTelOptions) Apply() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	// 同步 slog 选项
	o.syncSlogOptions()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 根据启用的信号初始化提供程序
	if err := o.initTraces(ctx); err != nil {
		return fmt.Errorf("failed to initialize traces: %w", err)
	}

	if err := o.initMetrics(ctx); err != nil {
		return fmt.Errorf("failed to initialize metrics: %w", err)
	}

	if err := o.initLogs(ctx); err != nil {
		return fmt.Errorf("failed to initialize logs: %w", err)
	}

	return nil
}

// Shutdown 优雅地关闭所有提供程序
func (o *OTelOptions) Shutdown(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	var errs []error

	errs = append(errs, o.shutdownProviders(ctx)...)

	// 关闭文件
	for _, file := range o.files {
		if err := file.Close(); err != nil {
			errs = append(errs, fmt.Errorf("file close: %w", err))
		}
	}
	o.files = o.files[:0] // 清空切片

	return errors.Join(errs...)
}

// GetTracerProvider 返回 tracer provider
func (o *OTelOptions) GetTracerProvider() *trace.TracerProvider {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.providers.tracer
}

// GetMeterProvider 返回 meter provider
func (o *OTelOptions) GetMeterProvider() *metric.MeterProvider {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.providers.meter
}

// GetLoggerProvider 返回 logger provider
func (o *OTelOptions) GetLoggerProvider() *log.LoggerProvider {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.providers.logger
}

func shutdownProvider(ctx context.Context, name string, provider Provider) error {
	if err := provider.Shutdown(ctx); err != nil {
		return fmt.Errorf("%s shutdown: %w", name, err)
	}
	return nil
}

func (o *OTelOptions) shutdownProviders(ctx context.Context) []error {
	var errs []error
	if o.providers.tracer != nil {
		errs = append(errs, shutdownProvider(ctx, "tracer", o.providers.tracer))
	}
	if o.providers.meter != nil {
		errs = append(errs, shutdownProvider(ctx, "meter", o.providers.meter))
	}
	if o.providers.logger != nil {
		errs = append(errs, shutdownProvider(ctx, "logger", o.providers.logger))
	}
	return errs
}
