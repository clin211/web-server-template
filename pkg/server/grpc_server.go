package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	genericoptions "github.com/clin211/gin-enterprise-template/pkg/options"
)

var errGRPCOptionsRequired = errors.New("gRPC options are required")

// GRPCServer 代表一个 GRPC 服务器.
type GRPCServer struct {
	srv *grpc.Server
	lis net.Listener
}

// NewGRPCServer 创建一个新的 GRPC 服务器实例.
func NewGRPCServer(
	grpcOptions *genericoptions.GRPCOptions,
	tlsOptions *genericoptions.TLSOptions,
	serverOptions []grpc.ServerOption,
	registerBuilder func() (func(grpc.ServiceRegistrar), string),
) (*GRPCServer, error) {

	if grpcOptions == nil {
		return nil, errGRPCOptionsRequired
	}
	listenConfig := net.ListenConfig{}
	lis, err := listenConfig.Listen(context.Background(), "tcp", grpcOptions.Addr)
	if err != nil {
		slog.Error("Failed to listen", "err", err)
		return nil, err
	}

	if tlsOptions != nil && tlsOptions.UseTLS {
		tlsConfig, err := tlsOptions.TLSConfig()
		if err != nil {
			if closeErr := lis.Close(); closeErr != nil {
				return nil, fmt.Errorf("close gRPC listener after TLS failure: %w", errors.Join(err, closeErr))
			}
			return nil, fmt.Errorf("create gRPC TLS config: %w", err)
		}
		serverOptions = append(serverOptions, grpc.Creds(credentials.NewTLS(tlsConfig)))
	}

	grpcsrv := grpc.NewServer(serverOptions...)

	registerFn, serverName := registerBuilder()
	registerFn(grpcsrv)
	registerHealthServer(serverName, grpcsrv)
	reflection.Register(grpcsrv)

	return &GRPCServer{
		srv: grpcsrv,
		lis: lis,
	}, nil
}

// Run 启动 GRPC 服务器，并将服务错误返回给调用方。
func (s *GRPCServer) Run() error {
	slog.Info("Start to listening the incoming requests",
		"protocol", "grpc",
		"addr", s.lis.Addr().String())
	if err := s.srv.Serve(s.lis); err != nil {
		return fmt.Errorf("serve gRPC server: %w", err)
	}
	return nil
}

// GracefulStop 优雅地关闭 GRPC 服务器.
func (s *GRPCServer) GracefulStop(ctx context.Context) error {
	slog.Info("Gracefully stop grpc server")
	done := make(chan struct{})
	go func() {
		s.srv.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		s.srv.Stop()
		return fmt.Errorf("gRPC graceful stop deadline exceeded: %w", ctx.Err())
	}
}

// registerHealthServer 注册健康检查服务.
func registerHealthServer(serverName string, grpcsrv *grpc.Server) {
	// 创建健康检查服务实例
	healthServer := health.NewServer()

	// 设定服务的健康状态
	healthServer.SetServingStatus(serverName, grpc_health_v1.HealthCheckResponse_SERVING)

	// 注册健康检查服务
	grpc_health_v1.RegisterHealthServer(grpcsrv, healthServer)
}
