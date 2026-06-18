package apiserver

import (
	"context"
	"net/http"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/handler"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/metrics"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	mw "github.com/clin211/gin-enterprise-template/internal/pkg/middleware/gin"
	"github.com/clin211/gin-enterprise-template/pkg/errorsx"
	genericmw "github.com/clin211/gin-enterprise-template/pkg/middleware/gin"
	"github.com/clin211/gin-enterprise-template/pkg/server"
	"github.com/gin-contrib/pprof"
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

	// 注册全局中间件，用于恢复 panic、设置 HTTP 头、添加请求 ID 等
	engine.Use(
		gin.Recovery(),
		mw.NoCache,
		mw.Cors,
		mw.Secure,
		otelgin.Middleware(serviceName, otelgin.WithFilter(func(rq *http.Request) bool {
			// 返回 false 表示不创建 span（过滤掉）
			return rq.URL.Path != "/metrics"
		})),
		genericmw.Observability(),
		mw.Context(),
	)

	// 注册 REST API 路由
	c.InstallRESTAPI(engine)

	httpsrv := server.NewHTTPServer(c.HTTPOptions, c.TLSOptions, engine)

	return &ginServer{srv: httpsrv}, nil
}

// 注册 API 路由。路由的路径和 HTTP 方法，严格遵循 REST 规范。
func (c *ServerConfig) InstallRESTAPI(engine *gin.Engine) {
	// 注册业务无关的 API 接口
	InstallGenericAPI(engine, c.resolveServiceName())

	// 认证中间件
	authMiddlewares := []gin.HandlerFunc{mw.AuthnMiddleware(c.retriever)}

	// 创建核心业务处理器
	hdl := handler.NewHandler(c.biz, c.val, authMiddlewares...)
	// 注册健康检查接口
	engine.GET("/healthz", hdl.Healthz)

	// 注册 v1 版本 API 路由分组
	v1 := engine.Group("/v1")
	// 注册用户登录、令牌刷新接口
	v1.POST("/auth/login", hdl.Login)
	// 注意：refresh-token 使用专门的 RefreshAuthnMiddleware，接受 refresh token
	v1.PUT("/auth/refresh-token", mw.RefreshAuthnMiddleware(c.retriever), hdl.RefreshToken)
	// 注册资源路由
	hdl.InstallAll(v1)
}

// InstallGenericAPI 注册业务无关的路由，例如 pprof、404 处理等。
// serviceName 用于 metrics 子系统的资源标识（OpenTelemetry resource attribute）。
func InstallGenericAPI(engine *gin.Engine, serviceName string) {
	// 注册 pprof 路由
	pprof.Register(engine)

	_ = metrics.Initialize(context.Background(), serviceName)

	// 暴露 /metrics 端点
	_ = engine.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// 注册 404 路由处理
	engine.NoRoute(func(c *gin.Context) {
		response := errorsx.FromBizError(errno.ErrPageNotFound)
		c.JSON(http.StatusNotFound, response)
	})
}

// RunOrDie 启动 Gin 服务器，出错则程序崩溃退出。
func (s *ginServer) RunOrDie() {
	s.srv.RunOrDie()
}

// GracefulStop 优雅停止服务器。
func (s *ginServer) GracefulStop(ctx context.Context) {
	s.srv.GracefulStop(ctx)
}
