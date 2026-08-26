package server

import (
	"context"
	"errors"
	"testing"
)

type serveTestServer struct {
	run     chan error
	stopErr error
}

func (s *serveTestServer) Run() error { return <-s.run }

func (s *serveTestServer) GracefulStop(context.Context) error {
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
