package app

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	flagHelp          = "help"
	flagHelpShorthand = "H"
)

func helpCommand(name string) *cobra.Command {
	return &cobra.Command{
		Use:   "help [command]",
		Short: "关于任何命令的帮助。",
		Long: `Help 为应用程序中的任何命令提供帮助。
只需输入 ` + name + ` help [命令路径] 即可获取完整详情。`,

		Run: func(c *cobra.Command, args []string) {
			cmd, _, e := c.Root().Find(args)
			if cmd == nil || e != nil {
				c.Printf("未知的帮助主题 %#q\n", args)
				_ = c.Root().Usage()
			} else {
				cmd.InitDefaultHelpFlag() // 使 'help' 标志能够显示
				_ = cmd.Help()
			}
		},
	}
}

// addHelpFlag 将特定应用程序的标志添加到指定的 FlagSet
// 对象。
func addHelpFlag(name string, fs *pflag.FlagSet) {
	fs.BoolP(flagHelp, flagHelpShorthand, false, fmt.Sprintf("%s 的帮助。", name))
}

// addHelpCommandFlag 将应用程序特定命令的标志添加到
// 指定的 FlagSet 对象。
func addHelpCommandFlag(usage string, fs *pflag.FlagSet) {
	fs.BoolP(flagHelp, flagHelpShorthand, false, fmt.Sprintf("%s 命令的帮助。", color.GreenString(strings.Split(usage, " ")[0])))
}
