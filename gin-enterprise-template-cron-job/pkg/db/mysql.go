package db

import (
	"fmt"
	"time"

	"database/sql"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// MySQLOptions 定义 MySQL 数据库的配置选项.
type MySQLOptions struct {
	Addr                  string
	Username              string
	Password              string
	Database              string
	MaxIdleConnections    int
	MaxOpenConnections    int
	MaxConnectionLifeTime time.Duration
	// +optional
	Logger logger.Interface
	// +optional
	// Location 指定时区，默认为 Local
	Location string
}

// DSN 从 MySQLOptions 返回数据源名称(DSN).
func (o *MySQLOptions) DSN() string {
	loc := o.Location
	if loc == "" {
		loc = "Local"
	}
	return fmt.Sprintf(`%s:%s@tcp(%s)/%s?charset=utf8&parseTime=%t&loc=%s`,
		o.Username,
		o.Password,
		o.Addr,
		o.Database,
		true,
		loc)
}

// NewMySQL 使用给定的选项创建一个新的 gorm 数据库实例.
func NewMySQL(opts *MySQLOptions) (*gorm.DB, error) {
	// 设置默认值以确保 opts 中的所有字段都可用.
	setMySQLDefaults(opts)

	db, err := gorm.Open(mysql.Open(opts.DSN()), &gorm.Config{
		// PrepareStmt 在缓存语句中执行给定的查询.
		// 这可以提高性能.
		PrepareStmt: true,
		Logger:      opts.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open mysql: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// SetMaxOpenConns 设置数据库的最大打开连接数.
	sqlDB.SetMaxOpenConns(opts.MaxOpenConnections)

	// SetConnMaxLifetime 设置连接可重用的最长时间.
	sqlDB.SetConnMaxLifetime(opts.MaxConnectionLifeTime)

	// SetMaxIdleConns 设置空闲连接池中的最大连接数.
	sqlDB.SetMaxIdleConns(opts.MaxIdleConnections)

	return db, nil
}

// setMySQLDefaults 为某些字段设置可用的默认值.
func setMySQLDefaults(opts *MySQLOptions) {
	if opts.Addr == "" {
		opts.Addr = "127.0.0.1:3306"
	}
	if opts.MaxIdleConnections == 0 {
		opts.MaxIdleConnections = 100
	}
	if opts.MaxOpenConnections == 0 {
		opts.MaxOpenConnections = 100
	}
	if opts.MaxConnectionLifeTime == 0 {
		opts.MaxConnectionLifeTime = time.Duration(10) * time.Second
	}
	if opts.Logger == nil {
		opts.Logger = logger.Default
	}
	if opts.Location == "" {
		opts.Location = "Local"
	}
}

// MustRawDB 获取底层的 *sql.DB，如果出错则 panic。
// 注意：此函数设计用于程序启动时的配置验证阶段，
// 此时如果发生错误通常表示配置严重错误，程序应该终止。
// 如果需要更安全的错误处理，请使用 db.DB() 方法。
func MustRawDB(db *gorm.DB) *sql.DB {
	raw, err := db.DB()
	if err != nil {
		panic(fmt.Errorf("failed to get raw DB: %w", err))
	}
	return raw
}

// RawDB 获取底层的 *sql.DB，返回错误。
func RawDB(db *gorm.DB) (*sql.DB, error) {
	raw, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get raw DB: %w", err)
	}
	return raw, nil
}
