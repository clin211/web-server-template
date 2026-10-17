package options

import (
	"errors"
	"github.com/spf13/pflag"
	"time"
)

var errInvalidTasksCapacityOrLeaseSettings = errors.New("invalid tasks capacity or lease settings")

// TaskOptions configures the single PostgreSQL task engine and its independent roles.
type TaskOptions struct {
	Enabled           bool          `mapstructure:"enabled"`
	Concurrency       int           `mapstructure:"concurrency"`
	PollInterval      time.Duration `mapstructure:"poll-interval"`
	LeaseDuration     time.Duration `mapstructure:"lease-duration"`
	HeartbeatInterval time.Duration `mapstructure:"heartbeat-interval"`
	DrainTimeout      time.Duration `mapstructure:"drain-timeout"`
	WorkerAddr        string        `mapstructure:"worker-addr"`
	SchedulerAddr     string        `mapstructure:"scheduler-addr"`
}

// NewTaskOptions returns bounded, deployable defaults.
func NewTaskOptions() *TaskOptions {
	return &TaskOptions{true, 4, time.Second, 30 * time.Second, 10 * time.Second, 30 * time.Second, "0.0.0.0:5560", "0.0.0.0:5561"}
}

// Validate rejects unsafe lease and shutdown bounds.
func (o *TaskOptions) Validate() []error {
	if o == nil || o.Concurrency < 1 || o.Concurrency > 128 || o.PollInterval <= 0 || o.LeaseDuration <= 0 || o.HeartbeatInterval <= 0 || o.HeartbeatInterval*3 > o.LeaseDuration || o.DrainTimeout <= 0 {
		return []error{errInvalidTasksCapacityOrLeaseSettings}
	}
	return nil
}

// AddFlags exposes role settings with the existing configuration precedence.
func (o *TaskOptions) AddFlags(f *pflag.FlagSet, prefix string) {
	f.BoolVar(&o.Enabled, prefix+".enabled", o.Enabled, "Run PostgreSQL tasks in Worker")
	f.IntVar(&o.Concurrency, prefix+".concurrency", o.Concurrency, "Maximum leased tasks per Worker")
	f.DurationVar(&o.PollInterval, prefix+".poll-interval", o.PollInterval, "Task and schedule poll interval")
	f.DurationVar(&o.LeaseDuration, prefix+".lease-duration", o.LeaseDuration, "Task lease duration")
	f.DurationVar(&o.HeartbeatInterval, prefix+".heartbeat-interval", o.HeartbeatInterval, "Lease renewal interval")
	f.DurationVar(&o.DrainTimeout, prefix+".drain-timeout", o.DrainTimeout, "Graceful drain limit")
	f.StringVar(&o.WorkerAddr, prefix+".worker-addr", o.WorkerAddr, "Worker health address")
	f.StringVar(&o.SchedulerAddr, prefix+".scheduler-addr", o.SchedulerAddr, "Scheduler health address")
}
