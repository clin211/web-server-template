package apiserver

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

type failingHTTPServer struct {
	err error

	mu      sync.Mutex
	stopped bool
}

type closeTracker struct {
	mu     sync.Mutex
	closed bool
}

func (c *closeTracker) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

func (c *closeTracker) wasClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

func (s *failingHTTPServer) Run() error { return s.err }

func (s *failingHTTPServer) GracefulStop(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopped = true
	return nil
}

func (s *failingHTTPServer) wasStopped() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stopped
}

func TestServerRunReturnsHTTPServerError(t *testing.T) {
	listenErr := errors.New("listen failed")
	httpServer := &failingHTTPServer{err: listenErr}
	srv := &Server{
		srv:      httpServer,
		async:    newAsyncRuntime(nil, nil),
		database: &closeTracker{},
	}

	err := srv.Run(context.Background())
	if !errors.Is(err, listenErr) {
		t.Fatalf("Run() error = %v, want wrapped %v", err, listenErr)
	}
	if !strings.Contains(err.Error(), "HTTP server failed") {
		t.Fatalf("Run() error = %q, want HTTP server context", err)
	}
	if !httpServer.wasStopped() {
		t.Fatal("GracefulStop() was not called after HTTP server failure")
	}
	if !srv.database.(*closeTracker).wasClosed() {
		t.Fatal("database was not closed after HTTP server failure")
	}
}
