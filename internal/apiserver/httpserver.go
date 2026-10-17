package apiserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/handler"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/metrics"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	"github.com/clin211/gin-enterprise-template/internal/pkg/localization"
	mw "github.com/clin211/gin-enterprise-template/internal/pkg/middleware/gin"
	"github.com/clin211/gin-enterprise-template/pkg/core"
	genericmw "github.com/clin211/gin-enterprise-template/pkg/middleware/gin"
	"github.com/clin211/gin-enterprise-template/pkg/server"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// ginServer 定义一个使用 Gin 框架开发的 HTTP 服务器.
type ginServer struct {
	srv server.Server
}

// 确保 *ginServer 实现了 server.Server 接口。
var _ server.Server = (*ginServer)(nil)

// fallbackServiceName 是当 OTelOptions 未配置 service name 时使用的兜底值；
// 与 pkg/options.NewOTelOptions 的默认值保持一致，避免 OTel/Prometheus 上报
// 出现空字符串导致后端拒绝。
const fallbackServiceName = "unknown-service"

// resolveServiceName 从 ServerConfig 的 OTel 配置中读取 service name，
// 任一层级为空时退化为 fallback，保证调用方拿到一个非空字符串。
func (c *ServerConfig) resolveServiceName() string {
	if c == nil || c.OTelOptions == nil || c.OTelOptions.ServiceName == "" {
		return fallbackServiceName
	}
	return c.OTelOptions.ServiceName
}

func (c *ServerConfig) NewGinServer() (*ginServer, error) {
	// 创建 Gin 引擎
	engine := gin.New()

	serviceName := c.resolveServiceName()
	var allowedOrigins []string
	if c.HTTPOptions != nil {
		allowedOrigins = c.HTTPOptions.AllowedOrigins
	}

	// 中间件顺序属于运行契约：先建立安全头和 request ID，再创建 OTel span、把关联
	// 字段写回 request context，最后由访问日志包围后续 handler。
	engine.Use(
		gin.Recovery(),
		mw.NoCache,
		mw.I18n(localization.New()),
		mw.CORS(allowedOrigins),
		mw.SecurityHeaders,
		mw.RequestIDMiddleware(),
		otelgin.Middleware(serviceName, otelgin.WithFilter(func(rq *http.Request) bool {
			// 返回 false 表示不创建 span（过滤掉）
			return rq.URL.Path != "/metrics"
		})),
		mw.Context(),
		genericmw.Observability(),
	)

	// 注册 REST API 路由
	if err := c.InstallRESTAPI(engine); err != nil {
		return nil, err
	}

	httpsrv, err := server.NewHTTPServer(c.HTTPOptions, c.TLSOptions, engine)
	if err != nil {
		return nil, fmt.Errorf("create HTTP server: %w", err)
	}

	return &ginServer{srv: httpsrv}, nil
}

// InstallRESTAPI 注册 API 路由。路由的路径和 HTTP 方法，严格遵循 REST 规范。
func (c *ServerConfig) InstallRESTAPI(engine *gin.Engine) error {
	// 注册业务无关的 API 接口
	if err := InstallGenericAPI(engine, c.resolveServiceName()); err != nil {
		return err
	}

	// 认证中间件
	authMiddlewares := []gin.HandlerFunc{mw.AuthnMiddleware(c.retriever, c.tokens)}

	// 创建核心业务处理器
	hdl := handler.NewHandler(c.biz, c.val, authMiddlewares...)
	hdl.SetReplayLimiter(c.loginLimiter)
	// 注册健康检查接口
	engine.GET("/healthz", hdl.Healthz)
	engine.GET("/readyz", c.Readyz)

	// 注册 v1 版本 API 路由分组
	v1 := engine.Group("/v1")
	// 注册用户登录、令牌刷新接口
	if c.LoginSecurityOptions != nil && c.LoginSecurityOptions.Enabled {
		v1.POST("/auth/login", mw.LoginRateLimitMiddleware(
			c.loginLimiter,
			c.LoginSecurityOptions.MaxAttempts,
			c.LoginSecurityOptions.Window,
		), hdl.Login)
	} else {
		v1.POST("/auth/login", hdl.Login)
	}
	// 注意：refresh-token 使用专门的 RefreshAuthnMiddleware，接受 refresh token
	v1.PUT("/auth/refresh-token", mw.RefreshAuthnMiddleware(c.retriever, c.tokens), hdl.RefreshToken)
	v1.GET("/auth/info", mw.AuthnMiddleware(c.retriever, c.tokens), hdl.Info)
	v1.POST("/auth/logout", mw.AuthnMiddleware(c.retriever, c.tokens), hdl.Logout)
	// 注册资源路由
	hdl.InstallAll(v1)
	return nil
}

// Readyz 检查服务处理业务请求所需的依赖是否可用。
func (c *ServerConfig) Readyz(ginContext *gin.Context) {
	if c == nil || c.readiness == nil {
		ginContext.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready"})
		return
	}
	checkCtx, cancel := context.WithTimeout(ginContext.Request.Context(), 2*time.Second)
	defer cancel()
	if err := c.readiness.Check(checkCtx); err != nil {
		slog.WarnContext(ginContext.Request.Context(), "Readiness check failed", "error", err)
		ginContext.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready"})
		return
	}
	ginContext.JSON(http.StatusOK, gin.H{"status": "ready"})
}

// InstallGenericAPI 注册业务无关的路由，例如 metrics、404 处理等。
// serviceName 用于 metrics 子系统的资源标识（OpenTelemetry resource attribute）。
func InstallGenericAPI(engine *gin.Engine, serviceName string) error {
	// metrics instruments 属于进程生命周期，不应绑定到任一请求或可取消的启动 context。
	if err := metrics.Initialize(context.Background(), serviceName); err != nil {
		return fmt.Errorf("initialize metrics: %w", err)
	}

	// 暴露 /metrics 端点
	_ = engine.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// 注册 404 路由处理
	engine.NoRoute(func(c *gin.Context) {
		core.WriteBizError(c, errno.ErrPageNotFound)
	})
	return nil
}

// Run 启动 Gin 服务器，并将启动或服务错误返回给调用方。
func (s *ginServer) Run() error {
	return s.srv.Run()
}

// GracefulStop 优雅停止服务器。
func (s *ginServer) GracefulStop(ctx context.Context) error {
	return s.srv.GracefulStop(ctx)
}
