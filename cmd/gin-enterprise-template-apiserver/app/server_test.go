package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	appoptions "github.com/clin211/gin-enterprise-template/cmd/gin-enterprise-template-apiserver/app/options"
)

type shutdownRecorder struct {
	err         error
	contextDone bool
	hasDeadline bool
}

func (r *shutdownRecorder) Shutdown(ctx context.Context) error {
	r.contextDone = ctx.Err() != nil
	_, r.hasDeadline = ctx.Deadline()
	return r.err
}

func TestShutdownProviderUsesFreshTimeoutContext(t *testing.T) {
	t.Parallel()

	recorder := &shutdownRecorder{}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	if err := shutdownProvider(canceled, recorder, time.Second); err != nil {
		t.Fatalf("shutdown provider: %v", err)
	}
	if recorder.contextDone {
		t.Error("Shutdown received an already canceled context")
	}
	if !recorder.hasDeadline {
		t.Error("Shutdown context has no deadline")
	}
}

func TestShutdownProviderWrapsError(t *testing.T) {
	t.Parallel()

	want := errors.New("flush failed")
	err := shutdownProvider(context.Background(), &shutdownRecorder{err: want}, time.Second)
	if !errors.Is(err, want) {
		t.Fatalf("shutdownProvider() error = %v, want wrapped %v", err, want)
	}
}

func TestLoadServerOptionsCommandLineOverridesConfig(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	opts := appoptions.NewServerOptions()
	cmd := &cobra.Command{}
	opts.AddFlags(cmd.PersistentFlags())
	if err := cmd.PersistentFlags().Set("jwt.secret", "command-line-secret-with-32-characters"); err != nil {
		t.Fatalf("set jwt flag: %v", err)
	}
	viper.SetConfigType("yaml")
	if err := viper.ReadConfig(strings.NewReader("jwt:\n  secret: file-secret-with-32-characters-long\n")); err != nil {
		t.Fatalf("read config: %v", err)
	}
	if err := loadServerOptions(cmd, opts); err != nil {
		t.Fatalf("loadServerOptions() error = %v", err)
	}
	if got := opts.JWTOptions.Secret; got != "command-line-secret-with-32-characters" {
		t.Fatalf("JWT secret = %q, want command-line value", got)
	}
}
