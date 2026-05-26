package apiserver

import (
	"context"
	"log/slog"
	"time"

	"github.com/clin211/gin-enterprise-template/pkg/authz"
	genericjob "github.com/clin211/gin-enterprise-template/pkg/job"
	genericoptions "github.com/clin211/gin-enterprise-template/pkg/options"
	"github.com/clin211/gin-enterprise-template/pkg/server"

	apiserverjobworker "github.com/clin211/gin-enterprise-template/internal/apiserver/job/worker"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
	"github.com/redis/go-redis/v9"
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
	JobOptions        *genericoptions.JobOptions
	// OTelOptions 提供 service name / endpoint 等可观测性配置；
	// httpserver 与 metrics 子系统从这里读取 service name，避免源码硬编码。
	OTelOptions *genericoptions.OTelOptions
}

// Server 表示 Web 服务器。
type Server struct {
	cfg         *ServerConfig
	srv         server.Server
	worker      *apiserverjobworker.Worker
	scheduler   *genericjob.Scheduler
	producer    genericjob.Producer
	redisClient *redis.Client
}

// ServerConfig 包含服务器的核心依赖和配置。
type ServerConfig struct {
	*Config
	biz       biz.IBiz
	val       *validation.Validator
	retriever mw.UserRetriever
	authz     *authz.Authz
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
	if s.worker != nil {
		if err := s.worker.Start(ctx); err != nil {
			return err
		}
	}
	if s.scheduler != nil {
		if err := s.scheduler.Start(ctx); err != nil {
			return err
		}
	}

	go s.srv.RunOrDie()

	<-ctx.Done()
	slog.Info("Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s.srv.GracefulStop(shutdownCtx)
	if s.scheduler != nil {
		s.scheduler.Stop(shutdownCtx)
	}
	if s.worker != nil {
		s.worker.Shutdown(shutdownCtx)
	}
	if s.producer != nil {
		if err := s.producer.Close(); err != nil {
			slog.WarnContext(shutdownCtx, "Failed to close job producer", "error", err)
		}
	}
	if s.redisClient != nil {
		if err := s.redisClient.Close(); err != nil {
			slog.WarnContext(shutdownCtx, "Failed to close redis client", "error", err)
		}
	}

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

// ProvideRedis 根据配置提供 redis 实例。
func ProvideRedis(cfg *Config) (*redis.Client, error) {
	if cfg.JobOptions == nil || (!cfg.JobOptions.Async.Enabled && !cfg.JobOptions.Scheduler.Enabled && !cfg.JobOptions.ClientTask.Enabled) {
		return redis.NewClient(&redis.Options{
			Addr:         cfg.RedisOptions.Addr,
			Username:     cfg.RedisOptions.Username,
			Password:     cfg.RedisOptions.Password,
			DB:           cfg.RedisOptions.Database,
			MaxRetries:   cfg.RedisOptions.MaxRetries,
			MinIdleConns: cfg.RedisOptions.MinIdleConns,
			DialTimeout:  cfg.RedisOptions.DialTimeout,
			ReadTimeout:  cfg.RedisOptions.ReadTimeout,
			WriteTimeout: cfg.RedisOptions.WriteTimeout,
			PoolTimeout:  cfg.RedisOptions.PoolTimeout,
			PoolSize:     cfg.RedisOptions.PoolSize,
		}), nil
	}
	return cfg.RedisOptions.NewClient()
}

// ProvideJobOptions 根据配置提供任务选项.
func ProvideJobOptions(cfg *Config) *genericoptions.JobOptions {
	return cfg.JobOptions
}

// NewWebServer 根据 ServerConfig 创建 Web 服务器.
func NewWebServer(serverConfig *ServerConfig) (server.Server, error) {
	return serverConfig.NewGinServer()
}
