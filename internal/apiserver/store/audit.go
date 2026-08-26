package store

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
)

// AuditLogStore 定义审计日志的最小写入接口。
type AuditLogStore interface {
	Create(ctx context.Context, record *model.AuditLogM) error
}

type auditLogStore struct{ core *datastore }

func newAuditLogStore(store *datastore) *auditLogStore { return &auditLogStore{core: store} }

func (s *auditLogStore) Create(ctx context.Context, record *model.AuditLogM) error {
	return s.core.DB(ctx).Create(record).Error
}
