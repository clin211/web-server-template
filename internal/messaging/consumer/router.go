package consumer

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/clin211/gin-enterprise-template/internal/messaging/outbox"
)

// ErrUnsupportedEventType 表示没有注册对应版本的事件处理器。
var ErrUnsupportedEventType = errors.New("unsupported event type")

var (
	errRouterEventTypeRequired = errors.New("event type is required")
	errRouterHandlerRequired   = errors.New("event handler is required")
	errRouterDuplicateType     = errors.New("event type is already registered")
)

// EventRouter 按 event_type 分发事件处理器。
type EventRouter struct {
	mu       sync.RWMutex
	handlers map[string]Handler
}

// NewEventRouter 创建空的事件路由器。
func NewEventRouter() *EventRouter {
	return &EventRouter{handlers: make(map[string]Handler)}
}

// HasHandlers reports whether at least one business event handler is registered.
// Runtime construction uses this to fail closed before starting a consumer with an
// accidentally empty routing table.
func (r *EventRouter) HasHandlers() bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.handlers) > 0
}

// Register 注册一个稳定版本的事件处理器。
func (r *EventRouter) Register(eventType string, handler Handler) error {
	if eventType == "" {
		return errRouterEventTypeRequired
	}
	if handler == nil {
		return errRouterHandlerRequired
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.handlers[eventType]; exists {
		return fmt.Errorf("%w: %s", errRouterDuplicateType, eventType)
	}
	r.handlers[eventType] = handler
	return nil
}

// Handle 分发事件；未知版本以 permanent error 返回。
func (r *EventRouter) Handle(ctx context.Context, envelope outbox.Envelope) error {
	// 锁只保护 handler map 的读取，调用前必须释放；慢 handler 不应阻塞注册流程，
	// handler 内部若触发注册也不会因锁重入而死锁。
	r.mu.RLock()
	handler := r.handlers[envelope.EventType]
	r.mu.RUnlock()
	if handler == nil {
		return Permanent(fmt.Errorf("%w: %s", ErrUnsupportedEventType, envelope.EventType))
	}
	return handler.Handle(ctx, envelope)
}
