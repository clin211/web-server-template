package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// Server 定义所有服务器类型的接口.
type Server interface {
	// RunOrDie 运行服务器，如果运行失败会退出程序（OrDie的含义所在）.
	RunOrDie()
	// GracefulStop 方法用来优雅关停服务器。关停服务器时需要处理 context 的超时时间.
	GracefulStop(ctx context.Context)
}

// Serve 启动服务器并阻塞，直到上下文被取消。
// 它确保在上下文完成时服务器被优雅关闭。
func Serve(ctx context.Context, srv Server) error {
	go srv.RunOrDie()

	// 阻塞直到上下文被取消或终止。
	<-ctx.Done()

	// 优雅关闭服务器。
	slog.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 优雅停止服务器。
	srv.GracefulStop(ctx)

	slog.Info("Server exited successfully.")

	return nil
}

// protocolName 从 http.Server 中获取协议名.
func protocolName(server *http.Server) string {
	if server.TLSConfig != nil {
		return "https"
	}
	return "http"
}
