package server

import (
	"context"
	"errors"
	"testing"
)

type serveTestServer struct {
	run         chan error
	stopErr     error
	stopContext context.Context
}

func (s *serveTestServer) Run() error { return <-s.run }

func (s *serveTestServer) GracefulStop(ctx context.Context) error {
	s.stopContext = ctx
	if s.stopErr == nil {
		s.run <- nil
	}
	return s.stopErr
}

func TestServePropagatesShutdownError(t *testing.T) {
	want := errors.New("shutdown failed")
	srv := &serveTestServer{run: make(chan error, 1), stopErr: want}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Serve(ctx, srv); !errors.Is(err, want) {
		t.Fatalf("Serve() error = %v, want %v", err, want)
	}
}

func TestServeWaitsForRunToExit(t *testing.T) {
	srv := &serveTestServer{run: make(chan error, 1)}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Serve(ctx, srv); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
}

func TestServePreservesShutdownContextValues(t *testing.T) {
	type contextKey struct{}
	srv := &serveTestServer{run: make(chan error, 1)}
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), contextKey{}, "trace-id"))
	cancel()
	if err := Serve(ctx, srv); err != nil {
		t.Fatal(err)
	}
	if got := srv.stopContext.Value(contextKey{}); got != "trace-id" {
		t.Fatalf("shutdown context value = %v", got)
	}
	if err := srv.stopContext.Err(); !errors.Is(err, context.Canceled) {
		t.Fatalf("shutdown context should be canceled when Serve returns: %v", err)
	}
}
