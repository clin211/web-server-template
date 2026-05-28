package options

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/spf13/pflag"
	"gorm.io/gorm"

	"github.com/clin211/gin-enterprise-template/pkg/db"
	gormlogger "github.com/clin211/gin-enterprise-template/pkg/logger/slog/gorm"
)

var _ IOptions = (*PostgreSQLOptions)(nil)

// PostgreSQLOptions 定义 postgresql 数据库的选项。
type PostgreSQLOptions struct {
	Addr                  string        `json:"addr,omitempty" mapstructure:"addr"`
	Username              string        `json:"username,omitempty" mapstructure:"username"`
	Password              string        `json:"-" mapstructure:"password"`
	Database              string        `json:"database" mapstructure:"database"`
	MaxIdleConnections    int           `json:"max-idle-connections,omitempty" mapstructure:"max-idle-connections,omitempty"`
	MaxOpenConnections    int           `json:"max-open-connections,omitempty" mapstructure:"max-open-connections"`
	MaxConnectionLifeTime time.Duration `json:"max-connection-life-time,omitempty" mapstructure:"max-connection-life-time"`
	LogLevel              int           `json:"log-level" mapstructure:"log-level"`
}

// NewPostgreSQLOptions 创建一个`零值`实例。
func NewPostgreSQLOptions() *PostgreSQLOptions {
	return &PostgreSQLOptions{
		Addr:                  "127.0.0.1:5432",
		Username:              "onex",
		Password:              "onex(#)666",
		Database:              "onex",
		MaxIdleConnections:    100,
		MaxOpenConnections:    100,
		MaxConnectionLifeTime: time.Duration(10) * time.Second,
		LogLevel:              1, // Silent
	}
}

// Validate 验证传递给 PostgreSQLOptions 的标志。
func (o *PostgreSQLOptions) Validate() []error {
	errs := []error{}

	// 仅在 Addr 配置了的情况下校验密码——这样不使用 PostgreSQL 的服务无须配置。
	if o.Addr != "" && IsPlaceholderSecret(o.Password) {
		errs = append(errs, fmt.Errorf(
			"postgresql.password looks like a placeholder/known-weak value (%q); please set a real password",
			o.Password,
		))
	}

	return errs
}

// AddFlags 将与特定 API 服务器的 postgresql 存储相关的标志添加到指定的 FlagSet。
func (o *PostgreSQLOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringVar(&o.Addr, fullPrefix+".addr", o.Addr, ""+
		"PostgreSQL service address. If left blank, the following related postgresql options will be ignored.")
	fs.StringVar(&o.Username, fullPrefix+".username", o.Username, "Username for access to postgresql service.")
	fs.StringVar(&o.Password, fullPrefix+".password", o.Password, ""+
		"Password for access to postgresql, should be used pair with password.")
	fs.StringVar(&o.Database, fullPrefix+".database", o.Database, ""+
		"Database name for the server to use.")
	fs.IntVar(&o.MaxIdleConnections, fullPrefix+".max-idle-connections", o.MaxOpenConnections, ""+
		"Maximum idle connections allowed to connect to postgresql.")
	fs.IntVar(&o.MaxOpenConnections, fullPrefix+".max-open-connections", o.MaxOpenConnections, ""+
		"Maximum open connections allowed to connect to postgresql.")
	fs.DurationVar(&o.MaxConnectionLifeTime, fullPrefix+".max-connection-life-time", o.MaxConnectionLifeTime, ""+
		"Maximum connection life time allowed to connect to postgresql.")
	fs.IntVar(&o.LogLevel, fullPrefix+".log-mode", o.LogLevel, ""+
		"Specify gorm log level.")
}

// NewDB 使用给定配置创建 postgresql 存储。
func (o *PostgreSQLOptions) NewDB() (*gorm.DB, error) {
	opts := &db.PostgreSQLOptions{
		Addr:                  o.Addr,
		Username:              o.Username,
		Password:              o.Password,
		Database:              o.Database,
		MaxIdleConnections:    o.MaxIdleConnections,
		MaxOpenConnections:    o.MaxOpenConnections,
		MaxConnectionLifeTime: o.MaxConnectionLifeTime,
		Logger:                gormlogger.New(slog.Default()),
	}

	return db.NewPostgreSQL(opts)
}
