package options

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/spf13/pflag"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var _ IOptions = (*MongoOptions)(nil)

// MongoOptions 包含连接到 MongoDB 服务器的选项。
type MongoOptions struct {
	URL        string        `json:"url" mapstructure:"url"`
	Database   string        `json:"database" mapstructure:"database"`
	Collection string        `json:"collection" mapstructure:"collection"`
	Username   string        `json:"username" mapstructure:"username"`
	Password   string        `json:"password" mapstructure:"password"`
	Timeout    time.Duration `json:"timeout" mapstructure:"timeout"`
	TLSOptions *TLSOptions   `json:"tls" mapstructure:"tls"`
}

// NewMongoOptions 创建一个`零值`实例。
func NewMongoOptions() *MongoOptions {
	return &MongoOptions{
		Timeout:    30 * time.Second,
		TLSOptions: NewTLSOptions(),
	}
}

// Validate 验证传递给 MongoOptions 的标志。
func (o *MongoOptions) Validate() []error {
	errs := []error{}

	if _, err := url.Parse(o.URL); err != nil {
		errs = append(errs, fmt.Errorf("unable to parse connection URL: %w", err))
	}

	if o.Database == "" {
		errs = append(errs, fmt.Errorf("--mongo.database can not be empty"))
	}

	if o.Collection == "" {
		errs = append(errs, fmt.Errorf("--mongo.collection can not be empty"))
	}

	if o.TLSOptions != nil {
		errs = append(errs, o.TLSOptions.Validate()...)
	}

	return errs
}

// AddFlags 将与特定 API 服务器的 redis 存储相关的标志添加到指定的 FlagSet。
func (o *MongoOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	o.TLSOptions.AddFlags(fs, fullPrefix+".tls")

	fs.DurationVar(&o.Timeout, fullPrefix+".timeout", o.Timeout, "Timeout is the maximum amount of time a dial will wait for a connect to complete.")
	fs.StringVar(&o.URL, fullPrefix+".url", o.URL, "The MongoDB server address.")
	fs.StringVar(&o.Database, fullPrefix+".database", o.Database, "The MongoDB database name.")
	fs.StringVar(&o.Collection, fullPrefix+".collection", o.Collection, "The MongoDB collection name.")
	fs.StringVar(&o.Username, fullPrefix+".username", o.Username, "Username of the MongoDB database (optional).")
	fs.StringVar(&o.Password, fullPrefix+".password", o.Password, "Password of the MongoDB database (optional).")
}

// NewClient 根据提供的选项创建新的 MongoDB 客户端。
func (o *MongoOptions) NewClient() (*mongo.Client, error) {
	// 设置客户端选项
	opts := options.Client().ApplyURI(o.URL).SetReadPreference(readpref.Primary())
	if o.Timeout > 0 {
		opts.SetConnectTimeout(o.Timeout).SetSocketTimeout(o.Timeout).SetServerSelectionTimeout(o.Timeout)
	}

	if o.Username != "" || o.Password != "" {
		opts.SetAuth(options.Credential{
			AuthSource: o.Database,
			Username:   o.Username,
			Password:   o.Password,
		})
	}

	if o.TLSOptions != nil {
		tlsConf, err := o.TLSOptions.TLSConfig()
		if err != nil {
			return nil, err
		}
		opts.SetTLSConfig(tlsConf)
	}

	ctx, cancel := context.WithTimeout(context.Background(), o.Timeout)
	defer cancel()

	// 连接到 MongoDB
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Ping MongoDB 服务器以检查连接
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, err
	}

	return client, nil
}
