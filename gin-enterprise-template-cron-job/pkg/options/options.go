package options

import "github.com/spf13/pflag"

// IOptions 定义实现通用选项的方法。
type IOptions interface {
	// Validate 验证所有必需的选项。
	// 如果需要，它也可以用于完成选项。
	Validate() []error

	// AddFlags 将所有选项字段作为命令行标志注册到给定的 FlagSet，
	// 直接使用提供的 fullPrefix。
	//
	// fullPrefix 应该是一个完整的前缀字符串，例如："onex.otel"。
	// 实现应将其自己的字段名称附加此前缀
	// 以构建最终的标志名称，例如：
	//   --onex.otel.endpoint
	//   --onex.otel.insecure
	AddFlags(fs *pflag.FlagSet, fullPrefix string)
}
