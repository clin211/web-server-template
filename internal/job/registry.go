package job

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
)

var errTrailingParams = errors.New("trailing params")
var errInvalidOrFrozenTaskRegistration = errors.New("invalid or frozen task registration")
var errDuplicateTaskRegistration = errors.New("duplicate task registration")
var errUnsupportedTaskTypeOrVersion = errors.New("unsupported task type or version")

// Result is the bounded output of a task handler.
type Result struct {
	Content     []byte
	ContentType string
}

// Handler executes outside a database transaction and must honor context cancellation.
type Handler interface {
	Validate(params []byte) error
	Execute(ctx context.Context, params []byte) (Result, error)
}
type typedHandler[P any] struct {
	validate func(P) error
	run      func(context.Context, P) (Result, error)
}

func (h typedHandler[P]) decode(b []byte) (P, error) {
	var p P
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(&p); err != nil {
		return p, Permanent(fmt.Errorf("decode task params: %w", err))
	}
	var extra any
	if err := d.Decode(&extra); !errors.Is(err, io.EOF) {
		return p, Permanent(errTrailingParams)
	}
	return p, h.validate(p)
}
func (h typedHandler[P]) Validate(b []byte) error { _, err := h.decode(b); return err }
func (h typedHandler[P]) Execute(ctx context.Context, b []byte) (Result, error) {
	p, err := h.decode(b)
	if err != nil {
		return Result{}, err
	}
	return h.run(ctx, p)
}

// Registry holds explicit immutable task registrations.
type Registry struct {
	mu       sync.RWMutex
	handlers map[string]Handler
	frozen   bool
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry { return &Registry{handlers: make(map[string]Handler)} }

// Register adapts a typed validator and handler; duplicate registrations fail.
func Register[P any](r *Registry, name string, version int, validate func(P) error, run func(context.Context, P) (Result, error)) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.frozen || name == "" || version < 1 || validate == nil || run == nil {
		return errInvalidOrFrozenTaskRegistration
	}
	key := fmt.Sprintf("%s:%d", name, version)
	if _, ok := r.handlers[key]; ok {
		return errDuplicateTaskRegistration
	}
	r.handlers[key] = typedHandler[P]{validate, run}
	return nil
}

// Freeze prevents mutations after startup.
func (r *Registry) Freeze() { r.mu.Lock(); defer r.mu.Unlock(); r.frozen = true }

// Get resolves an explicitly supported version.
func (r *Registry) Get(name string, v int) (Handler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h := r.handlers[fmt.Sprintf("%s:%d", name, v)]
	if h == nil {
		return nil, Permanent(errUnsupportedTaskTypeOrVersion)
	}
	return h, nil
}

// Validate checks the request against its handler contract.
func (r *Registry) Validate(name string, v int, b []byte) error {
	h, err := r.Get(name, v)
	if err != nil {
		return err
	}
	return h.Validate(b)
}
