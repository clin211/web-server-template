package job

import (
	"context"
	"errors"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"gorm.io/datatypes"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

type workerFake struct {
	mu         sync.Mutex
	claimed    bool
	finished   chan error
	heartbeats int
	canceled   bool
	lose       bool
}

func (f *workerFake) Claim(ctx context.Context, id string, d time.Duration) (*model.TaskM, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.claimed {
		return nil, nil
	}
	f.claimed = true
	return &model.TaskM{TaskID: "test", TaskType: "report.generate", ParamsVersion: 1, Params: datatypes.JSON(`{"days":1}`), TimeoutSeconds: 1}, nil
}
func (f *workerFake) Heartbeat(context.Context, *model.TaskM, time.Duration) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.heartbeats++
	if f.lose {
		return false, errors.New("lease lost")
	}
	return f.canceled, nil
}
func (f *workerFake) Finish(ctx context.Context, t *model.TaskM, r Result, e error) error {
	f.finished <- e
	return nil
}
func TestWorkerTimeoutPanicAndCancellation(t *testing.T) {
	for _, tc := range []struct {
		name            string
		panic, canceled bool
	}{{"timeout", false, false}, {"panic", true, false}, {"cancel", false, true}} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				f := &workerFake{finished: make(chan error, 1), canceled: tc.canceled}
				r := NewRegistry()
				if err := Register(r, "report.generate", 1, ValidateReport, func(ctx context.Context, p ReportParams) (Result, error) {
					if tc.panic {
						panic("private credential")
					}
					<-ctx.Done()
					return Result{}, ctx.Err()
				}); err != nil {
					t.Fatal(err)
				}
				w, err := NewWorker(f, r, WorkerOptions{ID: "fake", Concurrency: 1, Poll: time.Millisecond, Lease: time.Second, Heartbeat: 100 * time.Millisecond, Drain: time.Second})
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithCancel(context.Background())
				done := make(chan error, 1)
				go func() { done <- w.Run(ctx) }()
				runErr := <-f.finished
				if runErr == nil {
					t.Fatal("missing execution failure")
				}
				if !tc.panic {
					if tc.canceled && !errors.Is(runErr, context.Canceled) {
						t.Fatal(runErr)
					}
					if !tc.canceled && !errors.Is(runErr, context.DeadlineExceeded) {
						t.Fatal(runErr)
					}
				}
				cancel()
				if err := <-done; err != nil {
					t.Fatal(err)
				}
			})
		})
	}
}
func TestWorkerDrainsFetchedTask(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := &workerFake{finished: make(chan error, 1)}
		r := NewRegistry()
		started := make(chan struct{})
		release := make(chan struct{})
		if err := Register(r, "report.generate", 1, ValidateReport, func(ctx context.Context, p ReportParams) (Result, error) {
			close(started)
			<-release
			return Result{Content: []byte("done")}, nil
		}); err != nil {
			t.Fatal(err)
		}
		w, err := NewWorker(f, r, WorkerOptions{ID: "fake", Concurrency: 1, Poll: time.Millisecond, Lease: time.Second, Heartbeat: 100 * time.Millisecond, Drain: time.Second})
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- w.Run(ctx) }()
		<-started
		cancel()
		synctest.Wait()
		close(release)
		if err := <-f.finished; err != nil {
			t.Fatal(err)
		}
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	})
}
