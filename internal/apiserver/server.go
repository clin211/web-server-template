package apiserver

import (
	"context"
	"log/slog"
	"time"

	genericoptions "github.com/clin211/gin-enterprise-template/pkg/options"
	"github.com/clin211/gin-enterprise-template/pkg/server"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
	"gorm.io/gorm"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/biz"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/validation"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/known"
	mw "github.com/clin211/gin-enterprise-template/internal/pkg/middleware/gin"
	"github.com/clin211/gin-enterprise-template/pkg/token"
)

// Config 包含应用程序相关的配置。
type Config struct {
	JWTOptions        *genericoptions.JWTOptions
	TLSOptions        *genericoptions.TLSOptions
	HTTPOptions       *genericoptions.HTTPOptions
	PostgreSQLOptions *genericoptions.PostgreSQLOptions
	RedisOptions      *genericoptions.RedisOptions
	// OTelOptions 提供 service name / endpoint 等可观测性配置；
	// httpserver 与 metrics 子系统从这里读取 service name，避免源码硬编码。
	OTelOptions *genericoptions.OTelOptions
}

// Server 表示 Web 服务器。
type Server struct {
	cfg *ServerConfig
	srv server.Server
}

// ServerConfig 包含服务器的核心依赖和配置。
type ServerConfig struct {
	*Config
	biz       biz.IBiz
	val       *validation.Validator
	retriever mw.UserRetriever
}

// NewServer 初始化并返回一个新的 Server 实例。
func (cfg *Config) NewServer(ctx context.Context) (*Server, error) {
	where.RegisterTenant("user_id", func(ctx context.Context) string {
		return contextx.UserID(ctx)
	})

	// 初始化 token 包的签名密钥、Access Token 和 Refresh Token 过期时间
	token.Init(
		cfg.JWTOptions.Secret,
		cfg.JWTOptions.AccessExpiration,
		cfg.JWTOptions.RefreshExpiration,
		token.WithIdentityKey(known.XUserID),
	)
	// 创建核心服务器实例。
	return NewServer(cfg)
}

// Run 启动服务器并监听终止信号。
// 在收到终止信号时，它会优雅地关闭服务器。
func (s *Server) Run(ctx context.Context) error {
	go s.srv.RunOrDie()

	<-ctx.Done()
	slog.Info("Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s.srv.GracefulStop(shutdownCtx)

	slog.Info("Server exited successfully.")

	return nil
}

// NewDB 创建并返回一个用于 PostgreSQL 的 *gorm.DB 实例。
func (cfg *Config) NewDB() (*gorm.DB, error) {
	slog.Info("Initializing database connection", "type", "postgresql")
	db, err := cfg.PostgreSQLOptions.NewDB()
	if err != nil {
		slog.Error("Failed to create database connection", "error", err)
		return nil, err
	}

	// 自动迁移数据库模式
	// if err := registry.Migrate(db); err != nil {
	// 	slog.Error("Failed to migrate database schema", "error", err)
	// 	return nil, err
	// }

	return db, nil
}

// UserRetriever 定义一个用户数据获取器. 用来获取用户信息.
type UserRetriever struct {
	store store.IStore
}

// GetUser 根据用户 ID 获取用户信息.
func (r *UserRetriever) GetUser(ctx context.Context, userID string) (*model.UserM, error) {
	return r.store.User().Get(ctx, where.F("user_id", userID))
}

// ProvideDB 根据配置提供数据库实例。
func ProvideDB(cfg *Config) (*gorm.DB, error) {
	return cfg.NewDB()
}

// NewWebServer 根据 ServerConfig 创建 Web 服务器.
func NewWebServer(serverConfig *ServerConfig) (server.Server, error) {
	return serverConfig.NewGinServer()
}
