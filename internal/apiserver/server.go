package apiserver

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	OTelOptions          *genericoptions.OTelOptions
	KafkaOptions         *genericoptions.KafkaOptions
	OutboxOptions        *genericoptions.OutboxOptions
	ConsumerOptions      *genericoptions.ConsumerOptions
	LoginSecurityOptions *genericoptions.LoginSecurityOptions
}

// Server 表示 Web 服务器。
type Server struct {
	cfg      *ServerConfig
	srv      server.Server
	async    *AsyncRuntime
	database io.Closer
	redis    *token.RedisRefreshTokenStore
}

// ServerConfig 包含服务器的核心依赖和配置。
type ServerConfig struct {
	*Config
	biz          biz.IBiz
	val          *validation.Validator
	retriever    mw.UserRetriever
	readiness    readinessChecker
	tokens       *token.Manager
	loginLimiter mw.LoginRateLimiter
}

// NewServer 初始化并返回一个新的 Server 实例。
func (cfg *Config) NewServer(_ context.Context) (*Server, error) {
	// 创建核心服务器实例。
	server, _, err := NewServer(cfg)
	return server, err
}

// Run 启动服务器并监听终止信号。
// 在收到终止信号时，它会优雅地关闭服务器。
func (s *Server) Run(ctx context.Context) error {
	// 异步 runtime 先启动，避免 HTTP 已接受会产生 Outbox 的写请求，而 Relay/Consumer
	// 尚未进入工作状态。
	s.async.Start(ctx)
	httpDone := make(chan error, 1)
	go func() {
		httpDone <- s.srv.Run()
	}()

	var runErr error
	httpExited := false
	select {
	case <-ctx.Done():
	case err := <-httpDone:
		httpExited = true
		if err == nil {
			err = errors.New("HTTP server stopped unexpectedly")
		}
		runErr = fmt.Errorf("HTTP server failed: %w", err)
	case <-s.async.Done():
		// 异步链路提前退出属于服务级故障。立即停止 HTTP 接流，避免写请求继续产生
		// 无人发布的 Outbox 事件，同时把根因返回给进程管理器触发重启和告警。
		if err := s.async.Err(); err != nil {
			runErr = fmt.Errorf("async runtime failed: %w", err)
		}
	}
	slog.Info("Shutting down server...")
	// 根 context 收到信号后通常已经取消。关闭流程保留其中的 values、移除取消状态，
	// 再用独立 deadline 限制清理时间，否则 GracefulStop 会立即失败。
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()

	if err := s.shutdown(shutdownCtx); err != nil {
		runErr = errors.Join(runErr, fmt.Errorf("shutdown server resources: %w", err))
	}
	if !httpExited {
		select {
		case err := <-httpDone:
			if err != nil {
				runErr = errors.Join(runErr, fmt.Errorf("HTTP server stopped with error: %w", err))
			}
		case <-shutdownCtx.Done():
			runErr = errors.Join(runErr, fmt.Errorf("wait for HTTP server shutdown: %w", shutdownCtx.Err()))
		}
	}

	slog.Info("Server exited successfully.")

	return runErr
}

func (s *Server) shutdown(ctx context.Context) error {
	var serverErr error
	if s.srv != nil {
		if err := s.srv.GracefulStop(ctx); err != nil {
			serverErr = fmt.Errorf("stop HTTP server: %w", err)
		}
	}
	var asyncErr error
	if s.async != nil {
		if err := s.async.Shutdown(ctx); err != nil {
			asyncErr = fmt.Errorf("shutdown async runtime: %w", err)
		}
	}
	var databaseErr error
	if s.database != nil {
		if err := s.database.Close(); err != nil {
			databaseErr = fmt.Errorf("close database: %w", err)
		}
	}
	var redisErr error
	if s.redis != nil {
		if err := s.redis.Close(); err != nil {
			redisErr = fmt.Errorf("close redis: %w", err)
		}
	}
	return errors.Join(serverErr, asyncErr, databaseErr, redisErr)
}

// NewDB 创建并返回一个用于 PostgreSQL 的 *gorm.DB 实例。
func (cfg *Config) NewDB() (*gorm.DB, error) {
	slog.Info("Initializing database connection", "type", "postgresql")
	db, err := cfg.PostgreSQLOptions.NewDB()
	if err != nil {
		slog.Error("Failed to create database connection", "error", err)
		return nil, err
	}

	// 生产 schema 由部署阶段显式管理，服务启动时不执行 AutoMigrate，避免多副本同时
	// 运行 DDL 或应用在流量切换前擅自改变共享数据库结构。
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
func ProvideDB(cfg *Config) (*gorm.DB, func(), error) {
	db, err := cfg.NewDB()
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		sqlDB, sqlErr := db.DB()
		if sqlErr != nil {
			slog.Error("Failed to get database connection pool during constructor cleanup", "error", sqlErr)
			return
		}
		if closeErr := sqlDB.Close(); closeErr != nil {
			slog.Error("Failed to close database during constructor cleanup", "error", closeErr)
		}
	}
	return db, cleanup, nil
}

type databaseCloser struct{ db *gorm.DB }

func (c databaseCloser) Close() error {
	sqlDB, err := c.db.DB()
	if err != nil {
		return fmt.Errorf("get database connection pool: %w", err)
	}
	return sqlDB.Close()
}

func ProvideDatabaseCloser(db *gorm.DB) io.Closer {
	return databaseCloser{db: db}
}

func ProvideRedisRefreshTokenStore(cfg *Config) (*token.RedisRefreshTokenStore, func(), error) {
	if cfg == nil || cfg.RedisOptions == nil || cfg.RedisOptions.Addr == "" {
		return nil, func() {}, nil
	}
	client, err := cfg.RedisOptions.NewClient()
	if err != nil {
		return nil, nil, fmt.Errorf("create redis refresh token store: %w", err)
	}
	store := token.NewRedisRefreshTokenStore(client)
	return store, func() {
		if closeErr := store.Close(); closeErr != nil {
			slog.Error("Failed to close Redis during constructor cleanup", "error", closeErr)
		}
	}, nil
}

func ProvideTokenManagerWithStore(cfg *Config, store *token.RedisRefreshTokenStore) *token.Manager {
	return token.NewManagerWithRefreshStore(store, cfg.JWTOptions.Secret, cfg.JWTOptions.AccessExpiration, cfg.JWTOptions.RefreshExpiration, token.WithIdentityKey(known.XUserID))
}

func ProvideLoginRateLimiter(store *token.RedisRefreshTokenStore) mw.LoginRateLimiter {
	return store
}

type readinessChecker interface {
	Check(ctx context.Context) error
}

type redisReadinessChecker interface {
	Ping(ctx context.Context) error
}

type dependencyReadinessChecker struct {
	db     *gorm.DB
	redis  redisReadinessChecker
	async  *AsyncRuntime
	config *Config
}

func (c dependencyReadinessChecker) Check(ctx context.Context) error {
	var errs []error
	sqlDB, err := c.db.DB()
	if err != nil {
		errs = append(errs, fmt.Errorf("get database connection pool: %w", err))
	} else if err := sqlDB.PingContext(ctx); err != nil {
		errs = append(errs, fmt.Errorf("ping database: %w", err))
	}
	if c.redis != nil {
		if err := c.redis.Ping(ctx); err != nil {
			errs = append(errs, fmt.Errorf("check Redis: %w", err))
		}
	}
	if c.async != nil {
		if err := c.async.Check(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if kafkaEnabled(c.config) {
		if err := checkKafka(ctx, c.config.KafkaOptions); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func ProvideReadinessChecker(cfg *Config, db *gorm.DB, redisStore *token.RedisRefreshTokenStore, async *AsyncRuntime) readinessChecker {
	return dependencyReadinessChecker{db: db, redis: redisStore, async: async, config: cfg}
}

func kafkaEnabled(cfg *Config) bool {
	return cfg != nil && ((cfg.OutboxOptions != nil && cfg.OutboxOptions.Enabled) ||
		(cfg.ConsumerOptions != nil && cfg.ConsumerOptions.Enabled))
}

func checkKafka(ctx context.Context, options *genericoptions.KafkaOptions) error {
	if options == nil || len(options.Brokers) == 0 {
		return errors.New("Kafka brokers are not configured")
	}
	dialer, err := options.Dialer()
	if err != nil {
		return fmt.Errorf("create Kafka readiness dialer: %w", err)
	}
	var errs []error
	for _, broker := range options.Brokers {
		connection, dialErr := dialer.DialContext(ctx, "tcp", broker)
		if dialErr == nil {
			if closeErr := connection.Close(); closeErr != nil {
				return fmt.Errorf("close Kafka readiness connection: %w", closeErr)
			}
			return nil
		}
		errs = append(errs, fmt.Errorf("dial Kafka broker %s: %w", broker, dialErr))
	}
	return errors.Join(errs...)
}

// NewWebServer 根据 ServerConfig 创建 Web 服务器.
func NewWebServer(serverConfig *ServerConfig) (server.Server, error) {
	return serverConfig.NewGinServer()
}
