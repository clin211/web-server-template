package otelslog // import "go.opentelemetry.io/contrib/bridges/otelslog"

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"slices"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

// NewLogger 返回一个由新的 [Handler] 支持的 [slog.Logger]。有关
// 支持的 Handler 如何创建的详细信息，请参阅 [NewHandler]。
func NewLogger(name string, options ...Option) *slog.Logger {
	return slog.New(NewHandler(name, options...))
}

type config struct {
	provider   log.LoggerProvider
	version    string
	schemaURL  string
	attributes []attribute.KeyValue
	source     bool
	level      slog.Level
}

func newConfig(options []Option) config {
	var c config
	for _, opt := range options {
		c = opt.apply(c)
	}

	if c.provider == nil {
		c.provider = global.GetLoggerProvider()
	}

	return c
}

func (c config) logger(name string) log.Logger {
	var opts []log.LoggerOption
	if c.version != "" {
		opts = append(opts, log.WithInstrumentationVersion(c.version))
	}
	if c.schemaURL != "" {
		opts = append(opts, log.WithSchemaURL(c.schemaURL))
	}
	if c.attributes != nil {
		opts = append(opts, log.WithInstrumentationAttributes(c.attributes...))
	}
	return c.provider.Logger(name, opts...)
}

// Option 配置一个 [Handler]。
type Option interface {
	apply(config) config
}

type optFunc func(config) config

func (f optFunc) apply(c config) config { return f(c) }

// WithVersion 返回一个 [Option]，用于配置 [Handler] 使用的
// [log.Logger] 的版本。该版本应该是被记录的包的版本。
func WithVersion(version string) Option {
	return optFunc(func(c config) config {
		c.version = version
		return c
	})
}

// WithSchemaURL 返回一个 [Option]，用于配置 [Handler] 使用的
// [log.Logger] 的语义约定 schema URL。schemaURL 应该是日志记录中
// 使用的语义约定的 schema URL。
func WithSchemaURL(schemaURL string) Option {
	return optFunc(func(c config) config {
		c.schemaURL = schemaURL
		return c
	})
}

// WithAttributes 返回一个 [Option]，用于配置 [Handler] 使用的
// [log.Logger] 的 instrumentation scope 属性。
func WithAttributes(attributes ...attribute.KeyValue) Option {
	return optFunc(func(c config) config {
		c.attributes = attributes
		return c
	})
}

// WithLoggerProvider 返回一个 [Option]，用于配置 [Handler] 用来
// 创建其 [log.Logger] 的 [log.LoggerProvider]。
//
// 默认情况下，如果未提供此选项，Handler 将使用全局 LoggerProvider。
func WithLoggerProvider(provider log.LoggerProvider) Option {
	return optFunc(func(c config) config {
		c.provider = provider
		return c
	})
}

// WithSource 返回一个 [Option]，用于配置 [Handler] 在日志属性中
// 包含日志记录的源位置。
func WithSource(source bool) Option {
	return optFunc(func(c config) config {
		c.source = source
		return c
	})
}

// WithLevel 返回一个 [Option]，用于配置 Handler 的最低日志级别。
// 只有级别等于或高于此级别的日志记录才会被处理。
func WithLevel(level slog.Level) Option {
	return optFunc(func(c config) config {
		c.level = level
		return c
	})
}

// WithLevelString 返回一个 [Option]，使用字符串表示配置 Handler 的
// 最低日志级别。支持的值有：
// "DEBUG"、"INFO"、"WARN"、"WARNING"、"ERROR"。
// 如果提供了不支持的级别，默认为 INFO 级别。
func WithLevelString(levelStr string) Option {
	level := parseLevelString(levelStr)
	return WithLevel(level)
}

// parseLevelString 将字符串转换为 slog.Level
func parseLevelString(levelStr string) slog.Level {
	switch strings.ToUpper(strings.TrimSpace(levelStr)) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		// 如果是未知级别，默认为 INFO
		return slog.LevelInfo
	}
}

// Handler 是一个 [slog.Handler]，将接收到的所有日志记录发送到
// OpenTelemetry。有关如何进行转换的信息，请参阅包文档。
type Handler struct {
	// 通过显式使其不可比较来确保前向兼容性。
	noCmp [0]func() //nolint:unused  // 这个确实被使用了。

	attrs  *kvBuffer
	group  *group
	logger log.Logger
	level  slog.Level // 添加最低日志级别��段

	source bool
}

// 编译时检查 *Handler 是否实现了 slog.Handler。
var _ slog.Handler = (*Handler)(nil)

// NewHandler 返回一个新的 [Handler] 用作 [slog.Handler]。
//
// 如果未提供 [WithLoggerProvider]，返回的 Handler 将使用
// 全局 LoggerProvider。
//
// 提供的 name 需要唯一标识被记录的代码。这通常是代码的包名。
// 如果 name 为空，[log.Logger] 实现可能会用默认值覆盖此值。
func NewHandler(name string, options ...Option) *Handler {
	cfg := newConfig(options)
	return &Handler{
		logger: cfg.logger(name),
		source: cfg.source,
		level:  cfg.level, // 设置最低日志级别
	}
}

// Handle 处理传递的记录。
func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	h.logger.Emit(ctx, h.convertRecord(record))
	return nil
}

func (h *Handler) convertRecord(r slog.Record) log.Record {
	var record log.Record
	record.SetTimestamp(r.Time)
	record.SetBody(log.StringValue(r.Message))

	const sevOffset = slog.Level(log.SeverityDebug) - slog.LevelDebug
	record.SetSeverity(log.Severity(r.Level + sevOffset))
	record.SetSeverityText(r.Level.String())

	if h.source {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		record.AddAttributes(
			log.String(string(semconv.CodeFilePathKey), f.File),
			log.String(string(semconv.CodeFunctionNameKey), f.Function),
			log.Int(string(semconv.CodeLineNumberKey), f.Line),
		)
	}

	if h.attrs.Len() > 0 {
		record.AddAttributes(h.attrs.KeyValues()...)
	}

	n := r.NumAttrs()
	if h.group != nil {
		if n > 0 {
			buf := newKVBuffer(n)
			r.Attrs(buf.AddAttr)
			record.AddAttributes(h.group.KeyValue(buf.KeyValues()...))
		} else {
			// 如果没有属性，Handler 不应输出组。
			g := h.group.NextNonEmpty()
			if g != nil {
				record.AddAttributes(g.KeyValue())
			}
		}
	} else if n > 0 {
		buf := newKVBuffer(n)
		r.Attrs(buf.AddAttr)
		record.AddAttributes(buf.KeyValues()...)
	}

	return record
}

// Enabled 如果 Handler 被启用以记录提供的上下文和级别，则返回 true。
// 否则，如果未启用，则返回 false。
func (h *Handler) Enabled(ctx context.Context, l slog.Level) bool {
	// 首先检查本地级别过滤
	if l < h.level {
		return false
	}

	const sevOffset = slog.Level(log.SeverityDebug) - slog.LevelDebug
	param := log.EnabledParameters{Severity: log.Severity(l + sevOffset)}
	return h.logger.Enabled(ctx, param)
}

// WithAttrs 返回基于 h 的新 [slog.Handler]，它将使用传递的 attrs 进行日志记录。
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h2 := *h
	if h2.group != nil {
		h2.group = h2.group.Clone()
		h2.group.AddAttrs(attrs)
	} else {
		if h2.attrs == nil {
			h2.attrs = newKVBuffer(len(attrs))
		} else {
			h2.attrs = h2.attrs.Clone()
		}
		h2.attrs.AddAttrs(attrs)
	}
	return &h2
}

// WithGroup 返回基于 h 的新 [slog.Handler]，它将在提供的名称组内
// 记录所有消息和属性。
func (h *Handler) WithGroup(name string) slog.Handler {
	h2 := *h
	h2.group = &group{name: name, next: h2.group}
	return &h2
}

// group 表示从 slog 接收的组。
type group struct {
	// name 是组的名称。
	name string
	// attrs 是与组关联的属性。
	attrs *kvBuffer
	// next 指向持有此组的下一个组。
	//
	// 组在 OpenTelemetry 中表示为 map 值类型。这意味着对于如下
	// 的 slog 组层次结构...
	//
	//   WithGroup("G").WithGroup("H").WithGroup("I")
	//
	// 对应的 OpenTelemetry 日志值类型将具有以下层次结构...
	//
	//   KeyValue{
	//     Key: "G",
	//     Value: []KeyValue{{
	//       Key: "H",
	//       Value: []KeyValue{{
	//         Key: "I",
	//         Value: []KeyValue{},
	//       }},
	//     }},
	//   }
	//
	// 当记录属性时（即 Info("msg", "key", "value") 或
	// WithAttrs("key", "value")），它们需要被添加到"叶子"组中。在
	// 上面的示例中，那就是组 "I"：
	//
	//   KeyValue{
	//     Key: "G",
	//     Value: []KeyValue{{
	//       Key: "H",
	//       Value: []KeyValue{{
	//         Key: "I",
	//         Value: []KeyValue{
	//           String("key", "value"),
	//         },
	//       }},
	//     }},
	//   }
	//
	// 因此，组被结构化为链表，"叶子"节点是列表的头部。按照上面的示例，
	// 组数据表示将是...
	//
	//   *group{"I", next: *group{"H", next: *group{"G"}}}
	next *group
}

// NextNonEmpty 返回 g 的链表中具有属性的下一个组（包括 g 本身）。
// 如果未找到组，则返回 nil。
func (g *group) NextNonEmpty() *group {
	if g == nil || g.attrs.Len() > 0 {
		return g
	}
	return g.next.NextNonEmpty()
}

// KeyValue 返回包含 kvs 的组 g 作为 [log.KeyValue]。返回的 KeyValue 的值
// 将为 [log.KindMap] 类型。
//
// 传递的 kvs 在返回值中呈现，但不添加到组中。
//
// 这不会检查 g。调用者有责任确保 g 非空或 kvs 非空，以返回有效的组表示
//（根据 slog）。
func (g *group) KeyValue(kvs ...log.KeyValue) log.KeyValue {
	// 假设已经对组 g 进行了检查（即非空）。
	out := log.Map(g.name, g.attrs.KeyValues(kvs...)...)
	g = g.next
	for g != nil {
		// 如果没有属性，Handler 不应输出组。
		if g.attrs.Len() > 0 {
			out = log.Map(g.name, g.attrs.KeyValues(out)...)
		}
		g = g.next
	}
	return out
}

// Clone 返回 g 的副本。
func (g *group) Clone() *group {
	if g == nil {
		return nil
	}
	g2 := *g
	g2.attrs = g2.attrs.Clone()
	return &g2
}

// AddAttrs 将 attrs 添加到 g。
func (g *group) AddAttrs(attrs []slog.Attr) {
	if g.attrs == nil {
		g.attrs = newKVBuffer(len(attrs))
	}
	g.attrs.AddAttrs(attrs)
}

type kvBuffer struct {
	data []log.KeyValue
}

func newKVBuffer(n int) *kvBuffer {
	return &kvBuffer{data: make([]log.KeyValue, 0, n)}
}

// Len 返回 b 持有的 [log.KeyValue] 的数量。
func (b *kvBuffer) Len() int {
	if b == nil {
		return 0
	}
	return len(b.data)
}

// Clone 返回 b 的副本。
func (b *kvBuffer) Clone() *kvBuffer {
	if b == nil {
		return nil
	}
	return &kvBuffer{data: slices.Clone(b.data)}
}

// KeyValues 返回追加到 b 持有的 [log.KeyValue] 的 kvs。
func (b *kvBuffer) KeyValues(kvs ...log.KeyValue) []log.KeyValue {
	if b == nil {
		return kvs
	}
	return append(b.data, kvs...)
}

// AddAttrs 将 attrs 添加到 b。
func (b *kvBuffer) AddAttrs(attrs []slog.Attr) {
	b.data = slices.Grow(b.data, len(attrs))
	for _, a := range attrs {
		_ = b.AddAttr(a)
	}
}

// AddAttr 将 attr 添加到 b 并返回 true。
//
// 这旨在传递给 [slog.Record] 的 AddAttributes 方法。
//
// 如果 attr 是具有空键的组，其值将被展平。
//
// 如果 attr 为空，它将被丢弃。
func (b *kvBuffer) AddAttr(attr slog.Attr) bool {
	if attr.Key == "" {
		if attr.Value.Kind() == slog.KindGroup {
			// Handler 应该内联具有空键的组的 Attrs。
			for _, a := range attr.Value.Group() {
				b.data = append(b.data, log.KeyValue{
					Key:   a.Key,
					Value: convert(a.Value),
				})
			}
			return true
		}

		if attr.Value.Any() == nil {
			// Handler 应该忽略空的 Attr。
			return true
		}
	}
	b.data = append(b.data, log.KeyValue{
		Key:   attr.Key,
		Value: convert(attr.Value),
	})
	return true
}

func convert(v slog.Value) log.Value {
	switch v.Kind() {
	case slog.KindAny:
		return convertValue(v.Any())
	case slog.KindBool:
		return log.BoolValue(v.Bool())
	case slog.KindDuration:
		return log.Int64Value(v.Duration().Nanoseconds())
	case slog.KindFloat64:
		return log.Float64Value(v.Float64())
	case slog.KindInt64:
		return log.Int64Value(v.Int64())
	case slog.KindString:
		return log.StringValue(v.String())
	case slog.KindTime:
		return log.Int64Value(v.Time().UnixNano())
	case slog.KindUint64:
		const maxInt64 = ^uint64(0) >> 1
		u := v.Uint64()
		if u > maxInt64 {
			return log.Float64Value(float64(u))
		}
		return log.Int64Value(int64(u))
	case slog.KindGroup:
		g := v.Group()
		buf := newKVBuffer(len(g))
		buf.AddAttrs(g)
		return log.MapValue(buf.data...)
	case slog.KindLogValuer:
		return convert(v.Resolve())
	default:
		// 尽可能优雅地处理此情况。
		//
		// 不要在这里 panic。如果添加了新的 slog.Kind，目标 here 是让开发人员
		// 首先发现这一点。对新类型的测试会发现这个格式错误的属性以及 panic。
		// 但是，让用户提问为什么他们的属性有"unhandled: "前缀比说他们的代码
		// 正在 panic 更可取。
		return log.StringValue(fmt.Sprintf("unhandled: (%s) %+v", v.Kind(), v.Any()))
	}
}
