package store

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/clin211/gin-enterprise-template/pkg/store/logger/empty"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
)

// DBProvider 定义了一个提供数据库连接的接口。
type DBProvider interface {
	// DB 返回给定上下文的数据库实例。
	DB(ctx context.Context, wheres ...where.Where) *gorm.DB
}

// Option 定义用于配置 Store 的函数类型。
type Option[T any] func(*Store[T])

// Store 表示一个具有日志记录功能的通用数据存储。
type Store[T any] struct {
	logger  Logger
	storage DBProvider
}

// WithLogger 返回一个 Option 函数，将提供的 Logger 设置到 Store 以用于日志记录。
func WithLogger[T any](logger Logger) Option[T] {
	return func(s *Store[T]) {
		s.logger = logger
	}
}

// NewStore 使用提供的 DBProvider 创建 Store 的新实例。
func NewStore[T any](storage DBProvider, logger Logger) *Store[T] {
	if logger == nil {
		logger = empty.NewLogger()
	}

	return &Store[T]{
		logger:  logger,
		storage: storage,
	}
}

// db 获取数据库实例并应用的提供的 where 条件。
func (s *Store[T]) db(ctx context.Context, wheres ...where.Where) *gorm.DB {
	dbInstance := s.storage.DB(ctx)
	for _, whr := range wheres {
		if whr != nil {
			dbInstance = whr.Where(dbInstance)
		}
	}
	return dbInstance
}

// Create 向数据库中插入一个新对象。
func (s *Store[T]) Create(ctx context.Context, obj *T) error {
	if err := s.db(ctx).Create(obj).Error; err != nil {
		s.logger.Error(ctx, err, "Failed to insert object into database", "object", obj)
		return err
	}
	return nil
}

// Update 修改数据库中的现有对象。
func (s *Store[T]) Update(ctx context.Context, obj *T) error {
	if err := s.db(ctx).Save(obj).Error; err != nil {
		s.logger.Error(ctx, err, "Failed to update object in database", "object", obj)
		return err
	}
	return nil
}

// Delete 根据提供的 where 选项从数据库中删除对象。
func (s *Store[T]) Delete(ctx context.Context, opts *where.Options) error {
	err := s.db(ctx, opts).Delete(new(T)).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error(ctx, err, "Failed to delete object from database", "conditions", opts)
		return err
	}
	return nil
}

// Get 根据提供的 where 选项从数据库中检索单个对象。
func (s *Store[T]) Get(ctx context.Context, opts *where.Options) (*T, error) {
	var obj T
	if err := s.db(ctx, opts).First(&obj).Error; err != nil {
		s.logger.Error(ctx, err, "Failed to retrieve object from database", "conditions", opts)
		return nil, err
	}
	return &obj, nil
}

// List 根据提供的 where 选项从数据库中检索对象列表。
func (s *Store[T]) List(ctx context.Context, opts *where.Options) (count int64, ret []*T, err error) {
	err = s.db(ctx, opts).Order("id desc").Find(&ret).Offset(-1).Limit(-1).Count(&count).Error
	if err != nil {
		s.logger.Error(ctx, err, "Failed to list objects from database", "conditions", opts)
	}
	return
}
