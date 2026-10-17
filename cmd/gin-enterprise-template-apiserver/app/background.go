package app

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/clin211/gin-enterprise-template/cmd/gin-enterprise-template-apiserver/app/options"
	"github.com/clin211/gin-enterprise-template/internal/apiserver"
	jobruntime "github.com/clin211/gin-enterprise-template/internal/job/runtime"
	"github.com/clin211/gin-enterprise-template/pkg/core"
	"github.com/spf13/cobra"
)

// NewBackgroundCommand shares configuration with independently running workers and schedulers.
func NewBackgroundCommand(role string) *cobra.Command {
	opts := options.NewServerOptions()
	cmd := newRoleCommand(role, opts)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runConfiguredCommand(cmd, opts, func(ctx context.Context, cfg *apiserver.Config) error {
			return runBackground(ctx, cfg, opts, role)
		})
	}
	return cmd
}

func newRoleCommand(role string, opts *options.ServerOptions) *cobra.Command {
	var path string
	cmd := &cobra.Command{Use: "gin-enterprise-template-" + role, SilenceUsage: true, Args: cobra.NoArgs}
	cmd.PersistentFlags().StringVarP(&path, "config", "c", "", "Configuration file")
	opts.AddFlags(cmd.PersistentFlags())
	cobra.OnInitialize(core.OnInitialize(&path, "APP", searchDirs(), defaultConfigName))
	return cmd
}

func runConfiguredCommand(cmd *cobra.Command, opts *options.ServerOptions, run func(context.Context, *apiserver.Config) error) error {
	if err := loadServerOptions(cmd, opts); err != nil {
		return err
	}
	if errs := opts.PostgreSQLOptions.Validate(); len(errs) > 0 {
		return errors.Join(errs...)
	}
	cfg, err := opts.Config()
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	return run(ctx, cfg)
}

func validateBackgroundOptions(opts *options.ServerOptions, role string) error {
	if role == "worker" && (opts.ConsumerOptions.Enabled || opts.OutboxOptions.Enabled) {
		if errs := opts.KafkaOptions.Validate(); len(errs) > 0 {
			return errors.Join(errs...)
		}
		if errs := opts.ConsumerOptions.Validate(); len(errs) > 0 {
			return errors.Join(errs...)
		}
		if errs := opts.OutboxOptions.Validate(); len(errs) > 0 {
			return errors.Join(errs...)
		}
	}
	return nil
}

func runBackground(ctx context.Context, cfg *apiserver.Config, opts *options.ServerOptions, role string) error {
	if errs := opts.TaskOptions.Validate(); len(errs) > 0 {
		return errors.Join(errs...)
	}
	if err := validateBackgroundOptions(opts, role); err != nil {
		return err
	}
	opts.OTelOptions.ServiceName = "gin-enterprise-template-" + role
	if err := opts.OTelOptions.Apply(); err != nil {
		return err
	}
	runErr := jobruntime.Run(ctx, cfg, role)
	return errors.Join(runErr, shutdownProvider(ctx, opts.OTelOptions, 10*time.Second))
}
