package db

import (
	"net"
	"net/url"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// PostgreSQLOptions 定义 PostgreSQL 数据库的配置选项.
type PostgreSQLOptions struct {
	Addr                  string
	Username              string
	Password              string
	Database              string
	MaxIdleConnections    int
	MaxOpenConnections    int
	MaxConnectionLifeTime time.Duration
	SSLMode               string
	// +optional
	Logger logger.Interface
}

// DSN 从 PostgreSQLOptions 返回数据源名称(DSN).
func (o *PostgreSQLOptions) DSN() string {
	host, port := splitPostgreSQLAddr(o.Addr)
	sslMode := o.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	query := url.Values{
		"sslmode":  {sslMode},
		"TimeZone": {"Asia/Shanghai"},
	}
	dsn := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(o.Username, o.Password),
		Host:     net.JoinHostPort(host, port),
		Path:     "/" + url.PathEscape(o.Database),
		RawQuery: query.Encode(),
	}
	return dsn.String()
}

func splitPostgreSQLAddr(addr string) (string, string) {
	addr = strings.TrimSpace(addr)
	if host, port, err := net.SplitHostPort(addr); err == nil {
		return host, port
	}
	if strings.HasPrefix(addr, "[") && strings.HasSuffix(addr, "]") {
		return strings.Trim(addr, "[]"), "5432"
	}
	if strings.Count(addr, ":") > 1 {
		return addr, "5432"
	}
	if addr == "" {
		return "127.0.0.1", "5432"
	}
	return addr, "5432"
}

// NewPostgreSQL 使用给定的选项创建一个新的 gorm 数据库实例.
func NewPostgreSQL(opts *PostgreSQLOptions) (*gorm.DB, error) {
	// 设置默认值以确保 opts 中的所有字段都可用.
	setPostgreSQLDefaults(opts)

	db, err := gorm.Open(postgres.Open(opts.DSN()), &gorm.Config{
		// PrepareStmt 在缓存语句中执行给定的查询.
		// 这可以提高性能.
		PrepareStmt: true,
		Logger:      opts.Logger,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// SetMaxOpenConns 设置数据库的最大打开连接数.
	sqlDB.SetMaxOpenConns(opts.MaxOpenConnections)

	// SetConnMaxLifetime 设置连接可重用的最长时间.
	sqlDB.SetConnMaxLifetime(opts.MaxConnectionLifeTime)

	// SetMaxIdleConns 设置空闲连接池中的最大连接数.
	sqlDB.SetMaxIdleConns(opts.MaxIdleConnections)

	return db, nil
}

// setPostgreSQLDefaults 为某些字段设置可用的默认值.
func setPostgreSQLDefaults(opts *PostgreSQLOptions) {
	if opts.Addr == "" {
		opts.Addr = "127.0.0.1:5432"
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
	if opts.SSLMode == "" {
		opts.SSLMode = "disable"
	}
	if opts.Logger == nil {
		opts.Logger = logger.Default
	}
}
