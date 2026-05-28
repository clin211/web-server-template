package app

import (
	"github.com/spf13/pflag"
	cliflag "k8s.io/component-base/cli/flag"
)

// OptionsValidator 提供完成和验证选项的方法。
// 任何需要选项验证的组件都应该实现此接口。
type OptionsValidator interface {
	// Complete 完成所有必需的选项。
	Complete() error

	// Validate 验证所有必需的选项。
	Validate() error
}

// NamedFlagSetOptions 提供对服务器特定标志集的访问并嵌入
// 验证功能。
type NamedFlagSetOptions interface {
	// Flags 通过节段名称返回特定服务器的标志。
	Flags() cliflag.NamedFlagSets

	OptionsValidator
}

// FlagSetOptions 定义了可以添加到标志集并执行验证的
// 命令行选项接口。
type FlagSetOptions interface {
	// AddFlags 将命令特定的标志添加到提供的标志集。
	AddFlags(fs *pflag.FlagSet)

	OptionsValidator
}
