package otelslog_test

import (
	"go.opentelemetry.io/otel/log/noop"

	"go.opentelemetry.io/contrib/bridges/otelslog"
)

func Example() {
	// 改为使用可工作的 LoggerProvider 实现，例如使用 go.opentelemetry.io/otel/sdk/log。
	provider := noop.NewLoggerProvider()

	// 创建一个 *slog.Logger 并在您的应用程序中使用它。
	otelslog.NewLogger("my/pkg/name", otelslog.WithLoggerProvider(provider))
}
