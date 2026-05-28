package options

import (
	"errors"
	"time"

	"github.com/spf13/pflag"
)

// 确保接口实现
var _ IOptions = (*WatchOptions)(nil)

// WatchOptions 结构保存创建和运行监视服务器所需的配置选项。
type WatchOptions struct {
	// LockName 指定服务器使用的锁名称。
	LockName string `json:"lock-name" mapstructure:"lock-name"`

	// HealthzPort 是健康检查端口的端口号。
	HealthzPort int `json:"healthz-port" mapstructure:"healthz-port"`

	// DisableWatchers 是服务器运行时将被禁用的监视器列表。
	DisableWatchers []string `json:"disable-watchers" mapstructure:"disable-watchers"`

	// MaxWorkers 定义每个监视器可以生成的最大并发工作线程数。
	MaxWorkers int64 `json:"max-workers" mapstructure:"max-workers"`

	// WatchTimeout 定义每次单独监视执行的超时持续时间。
	WatchTimeout time.Duration `json:"watch-timeout" mapstructure:"watch-timeout"`

	// PerConcurrency 定义每个单独监视器允许的最大并发执行数。
	PerConcurrency int `json:"per-watch-concurrency" mapstructure:"per-watch-concurrency"`
}

// NewWatchOptions 初始化并返回具有默认值的新 WatchOptions 实例。
func NewWatchOptions() *WatchOptions {
	o := &WatchOptions{
		LockName:        "default-distributed-watch-lock",
		HealthzPort:     8881,
		DisableWatchers: []string{},
		MaxWorkers:      1000,
		WatchTimeout:    30 * time.Second,
		PerConcurrency:  10,
	}

	return o
}

// AddFlags 将与 WatchOptions 结构关联的命令行标志添加到提供的 FlagSet。
// 这将允许用户通过命令行参数配置监视服务器。
func (o *WatchOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringVar(&o.LockName, fullPrefix+".lock-name", o.LockName,
		"The name of the lock used by the server.")

	fs.IntVar(&o.HealthzPort, fullPrefix+".healthz-port", o.HealthzPort,
		"The port number for the health check endpoint.")

	fs.StringSliceVar(&o.DisableWatchers, fullPrefix+".disable-watchers", o.DisableWatchers,
		"The list of watchers that should be disabled.")

	fs.Int64Var(&o.MaxWorkers, fullPrefix+".max-workers", o.MaxWorkers,
		"Specify the maximum concurrency worker of each watcher.")

	fs.DurationVar(&o.WatchTimeout, fullPrefix+".timeout", o.WatchTimeout,
		"The timeout duration for each individual watch execution (e.g., 30s, 2m, 1h).")

	fs.IntVar(&o.PerConcurrency, fullPrefix+".per-concurrency", o.PerConcurrency,
		"The maximum number of concurrent executions allowed for each individual watcher.")
}

// Validate 检查 WatchOptions 结构的必需配置并返回错误列表。
func (o *WatchOptions) Validate() []error {
	errs := []error{}

	// 验证 LockName
	if o.LockName == "" {
		errs = append(errs, errors.New("lock-name cannot be empty"))
	}

	// 验证 HealthzPort
	if o.HealthzPort <= 0 || o.HealthzPort > 65535 {
		errs = append(errs, errors.New("healthz-port must be between 1 and 65535"))
	}

	// 验证 MaxWorkers
	if o.MaxWorkers <= 0 {
		errs = append(errs, errors.New("max-workers must be greater than 0"))
	}

	// 验证 WatchTimeout
	if o.WatchTimeout <= 0 {
		errs = append(errs, errors.New("watch-timeout must be greater than 0"))
	}

	// 检查合理的超时边界（可选但推荐）
	if o.WatchTimeout > 24*time.Hour {
		errs = append(errs, errors.New("watch-timeout should not exceed 24 hours for practical reasons"))
	}

	// 验证 PerConcurrency
	if o.PerConcurrency <= 0 {
		errs = append(errs, errors.New("per-watch-concurrency must be greater than 0"))
	}

	// 检查合理的并发边界
	if o.PerConcurrency > 1000 {
		errs = append(errs, errors.New("per-watch-concurrency should not exceed 1000 for resource management"))
	}

	// 交叉验证：PerConcurrency 不应超过 MaxWorkers
	if int64(o.PerConcurrency) > o.MaxWorkers {
		errs = append(errs, errors.New("per-watch-concurrency cannot be greater than max-workers"))
	}

	// 验证 DisableWatchers（可选：检查重复项）
	watcherSet := make(map[string]bool)
	for _, watcher := range o.DisableWatchers {
		if watcher == "" {
			errs = append(errs, errors.New("disable-watchers cannot contain empty strings"))
			continue
		}
		if watcherSet[watcher] {
			errs = append(errs, errors.New("disable-watchers contains duplicate entries: "+watcher))
		}
		watcherSet[watcher] = true
	}

	return errs
}
