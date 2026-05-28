package options

import (
	"fmt"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
	"github.com/segmentio/kafka-go/snappy"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"

	stringsutil "github.com/clin211/gin-enterprise-template/pkg/util/strings"
)

var _ IOptions = (*KafkaOptions)(nil)

type logger struct {
	v int32
}

func (l *logger) Printf(format string, args ...any) {
	klog.V(klog.Level(l.v)).Infof(format, args...)
}

type WriterOptions struct {
	// 限制传递消息的最大尝试次数。
	//
	// 默认最多尝试 10 次。
	MaxAttempts int `mapstructure:"max-attempts"`

	// 在接收到生成请求的响应之前，需要分区副本确认的数量。
	// 默认为 -1，表示等待所有副本，大于 0 的值表示需要确认消息成功的副本数。
	//
	// 此版本的 kafka-go (v0.3) 不支持 0 个必需的确认，由于
	// 使用 Kafka 协议实现它的一些内部复杂性。如果您
	// 特别需要该功能，则需要升级到 v0.4。
	RequiredAcks int `mapstructure:"required-acks"`

	// 将此标志设置为 true 会导致 WriteMessages 方法永不阻塞。
	// 这也意味着错误将被忽略，因为调用者不会收到
	// 返回值。仅在您不关心消息是否写入 kafka 的保证时使用此选项。
	Async bool `mapstructure:"async"`

	// 限制在发送到分区之前缓冲的消息数量。
	//
	// 默认使用目标批处理大小为 100 条消息。
	BatchSize int `mapstructure:"batch-size"`

	// 不完整的消息批次刷新到 kafka 的时间限制。
	//
	// 默认至少每秒刷新一次。
	BatchTimeout time.Duration `mapstructure:"batch-timeout"`

	// 在发送到分区之前限制请求的最大字节大小。
	//
	// 默认使用 kafka 默认值 1048576。
	BatchBytes int `mapstructure:"batch-bytes"`
}

type ReaderOptions struct {
	// GroupID 保存可选的消费者组 ID。如果指定了 GroupID，则
	// 不应指定 Partition，例如 0
	GroupID string `mapstructure:"group-id"`

	// GroupTopics 允许指定多个主题，但只能与
	// GroupID 结合使用，因为它是消费者组功能。因此，如果
	// 设置了 GroupID，则必须定义 Topic 或 GroupTopics。
	// GroupTopics []string

	// 要从中读取消息的分区。可以分配 Partition 或 GroupID
	// 中的一个，但不能同时分配两者
	Partition int `mapstructure:"partition"`

	// 内部消息队列的容量，如果未设置，则默认为 100。
	QueueCapacity int `mapstructure:"queue-capacity"`

	// MinBytes 向代理指示消费者将接受的最小批次大小。
	// 从低量主题消费时设置较高的最小值可能会导致
	// 传递延迟，因为代理没有足够的数据来满足定义的最小值。
	//
	// 默认值：1
	MinBytes int `mapstructure:"min-bytes"`

	// MaxBytes 向代理指示消费者将接受的最大批次大小。
	// 代理将截断消息以满足此最大值，因此
	// 选择一个对于最大消息大小来说足够高的值。
	//
	// 默认值：1MB
	MaxBytes int `mapstructure:"max-bytes"`

	// 从 kafka 获取消息批次时等待新数据的最长时间。
	//
	// 默认值：10s
	MaxWait time.Duration `mapstructure:"max-wait"`

	// ReadBatchTimeout 等待从 kafka 消息批次获取消息的时间量。
	//
	// 默认值：10s
	ReadBatchTimeout time.Duration `mapstructure:"read-batch-timeout"`

	// ReadLagInterval 设置更新 reader 滞后的频率。
	// 将此字段设置为负值将禁用滞后报告。
	// ReadLagInterval time.Duration

	// HeartbeatInterval 设置 reader 向消费者
	// 组发送心跳更新的可选频率。
	//
	// 默认值：3s
	//
	// 仅在设置 GroupID 时使用
	HeartbeatInterval time.Duration `mapstructure:"heartbeat-interval"`

	// CommitInterval 指示偏移量提交到
	// 代理的间隔。如果为 0，则将同步处理提交。
	//
	// 默认值：0
	//
	// 仅在设置 GroupID 时使用
	CommitInterval time.Duration `mapstructure:"commit-interval"`

	// RebalanceTimeout 可选地设置协调器在重新平衡期间
	// 等待成员加入的时间长度。对于负载较高的 kafka 服务器，
	// 将此值设置得更高可能很有用。
	//
	// 默认值：30s
	//
	// 仅在设置 GroupID 时使用
	RebalanceTimeout time.Duration `mapstructure:"rebalance-timeout"`

	// StartOffset 确定消费者组在发现没有提交偏移量的分区时应
	// 从哪里开始消费。如果
	// 非零，则必须设置为 FirstOffset 或 LastOffset 之一。
	//
	// 默认值：FirstOffset
	//
	// 仅在设置 GroupID 时使用
	StartOffset int64 `mapstructure:"start-offset"`

	// 在传递错误之前将进行的最大尝试次数限制。
	//
	// 默认尝试 3 次。
	MaxAttempts int `mapstructure:"max-attempts"`
}

// KafkaOptions 定义 kafka 集群的选项。
// kafka-go reader 和 writer 的通用选项。
type KafkaOptions struct {
	// kafka-go reader 和 writer 通用选项
	Brokers       []string      `mapstructure:"brokers"`
	Topic         string        `mapstructure:"topic"`
	ClientID      string        `mapstructure:"client-id"`
	Timeout       time.Duration `mapstructure:"timeout"`
	TLSOptions    *TLSOptions   `mapstructure:"tls"`
	SASLMechanism string        `mapstructure:"mechanism"`
	Username      string        `mapstructure:"username"`
	Password      string        `mapstructure:"password"`
	Algorithm     string        `mapstructure:"algorithm"`
	Compressed    bool          `mapstructure:"compressed"`

	// kafka-go writer 选项
	WriterOptions WriterOptions `mapstructure:"writer"`

	// kafka-go reader 选项
	ReaderOptions ReaderOptions `mapstructure:"reader"`
}

// NewKafkaOptions 创建一个`零值`实例。
func NewKafkaOptions() *KafkaOptions {
	return &KafkaOptions{
		TLSOptions: NewTLSOptions(),
		Timeout:    3 * time.Second,
		WriterOptions: WriterOptions{
			RequiredAcks: 1,
			MaxAttempts:  10,
			Async:        true,
			BatchSize:    100,
			BatchTimeout: 1 * time.Second,
			BatchBytes:   1 * MiB,
		},
		ReaderOptions: ReaderOptions{
			QueueCapacity:     100,
			MinBytes:          1,
			MaxBytes:          1 * MiB,
			MaxWait:           10 * time.Second,
			ReadBatchTimeout:  10 * time.Second,
			HeartbeatInterval: 3 * time.Second,
			CommitInterval:    0 * time.Second,
			RebalanceTimeout:  30 * time.Second,
			StartOffset:       kafka.FirstOffset,
			MaxAttempts:       3,
		},
	}
}

// Validate 验证传递给 KafkaOptions 的标志。
func (o *KafkaOptions) Validate() []error {
	errs := []error{}

	if len(o.Brokers) == 0 {
		errs = append(errs, fmt.Errorf("kafka broker can not be empty"))
	}

	if !o.TLSOptions.UseTLS && o.SASLMechanism != "" {
		errs = append(errs, fmt.Errorf("SASL-Mechanism is setted but use_ssl is false"))
	}

	if !stringsutil.StringIn(strings.ToLower(o.SASLMechanism), []string{"plain", "scram", ""}) {
		errs = append(errs, fmt.Errorf("doesn't support '%s' SASL mechanism", o.SASLMechanism))
	}

	if o.Timeout <= 0 {
		errs = append(errs, fmt.Errorf("--kafka.timeout cannot be negative"))
	}

	if o.ReaderOptions.GroupID != "" && o.ReaderOptions.Partition != 0 {
		errs = append(errs, fmt.Errorf("either Partition or GroupID may be assigned, but not both"))
	}

	if o.WriterOptions.BatchTimeout <= 0 {
		errs = append(errs, fmt.Errorf("--kafka.writer.batch-timeout cannot be negative"))
	}

	errs = append(errs, o.TLSOptions.Validate()...)

	return errs
}

// AddFlags 将与特定 API 服务器的 redis 存储相关的标志添加到指定的 FlagSet。
func (o *KafkaOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	o.TLSOptions.AddFlags(fs, fullPrefix+".tls")

	fs.StringSliceVar(&o.Brokers, fullPrefix+".brokers", o.Brokers, "The list of brokers used to discover the partitions available on the kafka cluster.")
	fs.StringVar(&o.Topic, fullPrefix+".topic", o.Topic, "The topic that the writer/reader will produce/consume messages to.")
	fs.StringVar(&o.ClientID, fullPrefix+".client-id", o.ClientID, " Unique identifier for client connections established by this Dialer. ")
	fs.DurationVar(&o.Timeout, fullPrefix+".timeout", o.Timeout, "Timeout is the maximum amount of time a dial will wait for a connect to complete.")
	fs.StringVar(&o.SASLMechanism, fullPrefix+".mechanism", o.SASLMechanism, "Configures the Dialer to use SASL authentication.")
	fs.StringVar(&o.Username, fullPrefix+".username", o.Username, "Username of the kafka cluster.")
	fs.StringVar(&o.Password, fullPrefix+".password", o.Password, "Password of the kafka cluster.")
	fs.StringVar(&o.Algorithm, fullPrefix+".algorithm", o.Algorithm, "Algorithm used to create sasl.Mechanism.")
	fs.BoolVar(&o.Compressed, fullPrefix+".compressed", o.Compressed, "compressed is used to specify whether compress Kafka messages.")
	fs.IntVar(&o.WriterOptions.RequiredAcks, fullPrefix+".required-acks", o.WriterOptions.RequiredAcks, ""+
		"Number of acknowledges from partition replicas required before receiving a response to a produce request.")
	fs.IntVar(&o.WriterOptions.MaxAttempts, fullPrefix+".writer.max-attempts", o.WriterOptions.MaxAttempts, ""+
		"Limit on how many attempts will be made to deliver a message.")
	fs.BoolVar(&o.WriterOptions.Async, fullPrefix+".writer.async", o.WriterOptions.Async, "Limit on how many attempts will be made to deliver a message.")
	fs.IntVar(&o.WriterOptions.BatchSize, fullPrefix+".writer.batch-size", o.WriterOptions.BatchSize, ""+
		"Limit on how many messages will be buffered before being sent to a partition.")
	fs.DurationVar(&o.WriterOptions.BatchTimeout, fullPrefix+".writer.batch-timeout", o.WriterOptions.BatchTimeout, ""+
		"Time limit on how often incomplete message batches will be flushed to kafka.")
	fs.IntVar(&o.WriterOptions.BatchBytes, fullPrefix+".writer.batch-bytes", o.WriterOptions.BatchBytes, ""+
		"Limit the maximum size of a request in bytes before being sent to a partition.")
	fs.StringVar(&o.ReaderOptions.GroupID, fullPrefix+".reader.group-id", o.ReaderOptions.GroupID, ""+
		"GroupID holds the optional consumer group id. If GroupID is specified, then Partition should NOT be specified e.g. 0.")
	fs.IntVar(&o.ReaderOptions.Partition, fullPrefix+".reader.partition", o.ReaderOptions.Partition, "Partition to read messages from.")
	fs.IntVar(&o.ReaderOptions.QueueCapacity, fullPrefix+".reader.queue-capacity", o.ReaderOptions.QueueCapacity, ""+
		"The capacity of the internal message queue, defaults to 100 if none is set.")
	fs.IntVar(&o.ReaderOptions.MinBytes, fullPrefix+".reader.min-bytes", o.ReaderOptions.MinBytes, ""+
		"MinBytes indicates to the broker the minimum batch size that the consumer will accept.")
	fs.IntVar(&o.ReaderOptions.MaxBytes, fullPrefix+".reader.max-bytes", o.ReaderOptions.MaxBytes, ""+
		"MaxBytes indicates to the broker the maximum batch size that the consumer will accept.")
	fs.DurationVar(&o.ReaderOptions.MaxWait, fullPrefix+".reader.max-wait", o.ReaderOptions.MaxWait, ""+
		"Maximum amount of time to wait for new data to come when fetching batches of messages from kafka.")
	fs.DurationVar(&o.ReaderOptions.ReadBatchTimeout, fullPrefix+".reader.read-batch-timeout", o.ReaderOptions.ReadBatchTimeout, ""+
		"ReadBatchTimeout amount of time to wait to fetch message from kafka messages batch.")
	fs.DurationVar(&o.ReaderOptions.HeartbeatInterval, fullPrefix+".reader.heartbeat-interval", o.ReaderOptions.HeartbeatInterval, ""+
		"HeartbeatInterval sets the optional frequency at which the reader sends the consumer group heartbeat update.")
	fs.DurationVar(&o.ReaderOptions.CommitInterval, fullPrefix+".reader.commit-interval", o.ReaderOptions.CommitInterval, ""+
		"CommitInterval indicates the interval at which offsets are committed to the broker.")
	fs.DurationVar(&o.ReaderOptions.RebalanceTimeout, fullPrefix+".reader.rebalance-timeout", o.ReaderOptions.RebalanceTimeout, ""+
		"RebalanceTimeout optionally sets the length of time the coordinator will wait for members to join as part of a rebalance.")
	fs.Int64Var(&o.ReaderOptions.StartOffset, fullPrefix+".reader.start-offset", o.ReaderOptions.StartOffset, ""+
		"StartOffset determines from whence the consumer group should begin consuming when it finds a partition without a committed offset.")
	fs.IntVar(&o.ReaderOptions.MaxAttempts, fullPrefix+".reader.max-attempts", o.ReaderOptions.MaxAttempts, ""+
		"Limit of how many attempts will be made before delivering the error. ")
}

func (o *KafkaOptions) GetMechanism() (sasl.Mechanism, error) {
	var mechanism sasl.Mechanism

	switch o.SASLMechanism {
	case "":
		break
	case "PLAIN", "plain":
		mechanism = plain.Mechanism{Username: o.Username, Password: o.Password}
	case "SCRAM", "scram":
		algorithm := scram.SHA256
		if o.Algorithm == "sha-512" || o.Algorithm == "SHA-512" {
			algorithm = scram.SHA512
		}
		var err error
		mechanism, err = scram.Mechanism(algorithm, o.Username, o.Password)
		if err != nil {
			return nil, fmt.Errorf("failed initialize kafka mechanism: %w", err)
		}
	default:
	}

	return mechanism, nil
}

func (o *KafkaOptions) Dialer() (*kafka.Dialer, error) {
	tlsConfig, err := o.TLSOptions.TLSConfig()
	if err != nil {
		return nil, err
	}

	mechanism, err := o.GetMechanism()
	if err != nil {
		return nil, err
	}

	return &kafka.Dialer{
		Timeout:       o.Timeout,
		ClientID:      o.ClientID,
		TLS:           tlsConfig,
		SASLMechanism: mechanism,
	}, nil
}

func (o *KafkaOptions) Writer() (*kafka.Writer, error) {
	dialer, err := o.Dialer()
	if err != nil {
		return nil, err
	}

	// Kafka writer connection config
	config := kafka.WriterConfig{
		Brokers:      o.Brokers,
		Topic:        o.Topic,
		Balancer:     &kafka.LeastBytes{},
		Dialer:       dialer,
		WriteTimeout: o.Timeout,
		ReadTimeout:  o.Timeout,

		Async:        o.WriterOptions.Async,
		BatchSize:    o.WriterOptions.BatchSize,
		BatchBytes:   o.WriterOptions.BatchBytes,
		BatchTimeout: o.WriterOptions.BatchTimeout,
		MaxAttempts:  o.WriterOptions.MaxAttempts,
		Logger:       &logger{4},
		ErrorLogger:  &logger{1},
	}

	if o.Compressed {
		config.CompressionCodec = snappy.NewCompressionCodec()
	}

	kafkaWriter := kafka.NewWriter(config)
	return kafkaWriter, nil
}
