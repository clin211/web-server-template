package options

import (
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"

	"github.com/gorilla/mux"
	"github.com/spf13/pflag"
)

var _ IOptions = (*HealthOptions)(nil)

// HealthOptions 定义 redis 集群的选项。
type HealthOptions struct {
	// 通过公开分析信息来启用调试。
	HTTPProfile        bool   `json:"enable-http-profiler" mapstructure:"enable-http-profiler"`
	HealthCheckPath    string `json:"check-path" mapstructure:"check-path"`
	HealthCheckAddress string `json:"check-address" mapstructure:"check-address"`
}

// NewHealthOptions 创建一个`零值`实例。
func NewHealthOptions() *HealthOptions {
	return &HealthOptions{
		HTTPProfile:        false,
		HealthCheckPath:    "/healthz",
		HealthCheckAddress: "0.0.0.0:20250",
	}
}

// Validate 验证传递给 HealthOptions 的标志。
func (o *HealthOptions) Validate() []error {
	errs := []error{}

	return errs
}

// AddFlags 将与特定 API 服务器的 redis 存储相关的标志添加到指定的 FlagSet。
func (o *HealthOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.BoolVar(&o.HTTPProfile, fullPrefix+".enable-http-profiler", o.HTTPProfile, "Expose runtime profiling data via HTTP.")
	fs.StringVar(&o.HealthCheckPath, fullPrefix+".check-path", o.HealthCheckPath, "Specifies liveness health check request path.")
	fs.StringVar(&o.HealthCheckAddress, fullPrefix+".check-address", o.HealthCheckAddress, "Specifies liveness health check bind address.")
}

func (o *HealthOptions) ServeHealthCheck() {
	r := mux.NewRouter()

	r.HandleFunc(o.HealthCheckPath, handler).Methods(http.MethodGet)
	if o.HTTPProfile {
		r.HandleFunc("/debug/pprof/profile", pprof.Profile)
		r.HandleFunc("/debug/pprof/{_:.*}", pprof.Index)
	}

	slog.Info("Starting health check server", "path", o.HealthCheckPath, "addr", o.HealthCheckAddress)
	if err := http.ListenAndServe(o.HealthCheckAddress, r); err != nil {
		slog.Error("Error serving health check endpoint", "error", err)
		os.Exit(1)
	}
}

func handler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-type", "application/json")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(`{"status": "ok"}`))
}
