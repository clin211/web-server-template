package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Server 定义所有服务器类型的接口.
type Server interface {
	// Run 运行服务器，并将启动或服务错误返回给调用方。
	Run() error
	// GracefulStop 方法用来优雅关停服务器。关停服务器时需要处理 context 的超时时间.
	GracefulStop(ctx context.Context) error
}

// Serve 启动服务器并阻塞，直到上下文被取消。
// 它确保在上下文完成时服务器被优雅关闭。
func Serve(ctx context.Context, srv Server) error {
	done := make(chan error, 1)
	go func() {
		done <- srv.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			return err
		}
		return errors.New("server stopped unexpectedly")
	case <-ctx.Done():
	}

	// 优雅关闭服务器。
	slog.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 优雅停止服务器，并传播关闭失败，避免探针或进程管理器误判已经安全退出。
	shutdownErr := srv.GracefulStop(ctx)
	if shutdownErr != nil {
		return fmt.Errorf("gracefully stop server: %w", shutdownErr)
	}
	// GracefulStop 成功返回后 Run 应该已经结束；读取结果可以避免遗留 goroutine
	// 继续占用 listener 或连接资源。
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("server stopped with error: %w", err)
		}
	case <-ctx.Done():
		return fmt.Errorf("wait for server shutdown: %w", ctx.Err())
	}

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
