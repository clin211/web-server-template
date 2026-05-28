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

var _ IOptions = (*MySQLOptions)(nil)

// MySQLOptions 定义 mysql 数据库的选项。
type MySQLOptions struct {
	Addr                  string        `json:"addr,omitempty" mapstructure:"addr"`
	Username              string        `json:"username,omitempty" mapstructure:"username"`
	Password              string        `json:"-" mapstructure:"password"`
	Database              string        `json:"database" mapstructure:"database"`
	MaxIdleConnections    int           `json:"max-idle-connections,omitempty" mapstructure:"max-idle-connections,omitempty"`
	MaxOpenConnections    int           `json:"max-open-connections,omitempty" mapstructure:"max-open-connections"`
	MaxConnectionLifeTime time.Duration `json:"max-connection-life-time,omitempty" mapstructure:"max-connection-life-time"`
	LogLevel              int           `json:"log-level" mapstructure:"log-level"`
}

// NewMySQLOptions 创建一个`零值`实例。
func NewMySQLOptions() *MySQLOptions {
	return &MySQLOptions{
		Addr:                  "127.0.0.1:3306",
		Username:              "",
		Password:              "",
		Database:              "",
		MaxIdleConnections:    100,
		MaxOpenConnections:    100,
		MaxConnectionLifeTime: time.Duration(10) * time.Second,
		LogLevel:              1, // Silent
	}
}

// Validate 验证传递给 MySQLOptions 的标志。
func (o *MySQLOptions) Validate() []error {
	errs := []error{}

	return errs
}

// AddFlags 将与特定 API 服务器的 mysql 存储相关的标志添加到指定的 FlagSet。
func (o *MySQLOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringVar(&o.Addr, fullPrefix+".host", o.Addr, ""+
		"MySQL service host address.")
	fs.StringVar(&o.Username, fullPrefix+".username", o.Username, "Username for access to mysql service.")
	fs.StringVar(&o.Password, fullPrefix+".password", o.Password, ""+
		"Password for access to mysql, should be used pair with password.")
	fs.StringVar(&o.Database, fullPrefix+".database", o.Database, ""+
		"Database name for the server to use.")
	fs.IntVar(&o.MaxIdleConnections, fullPrefix+".max-idle-connections", o.MaxOpenConnections, ""+
		"Maximum idle connections allowed to connect to .")
	fs.IntVar(&o.MaxOpenConnections, fullPrefix+".max-open-connections", o.MaxOpenConnections, ""+
		"Maximum open connections allowed to connect to .")
	fs.DurationVar(&o.MaxConnectionLifeTime, fullPrefix+".max-connection-life-time", o.MaxConnectionLifeTime, ""+
		"Maximum connection life time allowed to connect to .")
	fs.IntVar(&o.LogLevel, fullPrefix+".log-mode", o.LogLevel, ""+
		"Specify gorm log level.")
}

// DSN 从 MySQLOptions 返回 DSN。
func (o *MySQLOptions) DSN() string {
	return fmt.Sprintf(`%s:%s@tcp(%s)/%s?charset=utf8&parseTime=%t&loc=%s`,
		o.Username,
		o.Password,
		o.Addr,
		o.Database,
		true,
		"Local")
}

// NewDB 使用给定配置创建 mysql 存储。
func (o *MySQLOptions) NewDB() (*gorm.DB, error) {
	opts := &db.MySQLOptions{
		Addr:                  o.Addr,
		Username:              o.Username,
		Password:              o.Password,
		Database:              o.Database,
		MaxIdleConnections:    o.MaxIdleConnections,
		MaxOpenConnections:    o.MaxOpenConnections,
		MaxConnectionLifeTime: o.MaxConnectionLifeTime,
		Logger:                gormlogger.New(slog.Default()),
	}

	return db.NewMySQL(opts)
}
