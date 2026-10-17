package app

import (
	"fmt"
	"strings"

	"github.com/clin211/gin-enterprise-template/pkg/i18n"
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
		Short: i18n.DefaultMessage("CLI.HelpShort"),
		Long:  fmt.Sprintf(i18n.DefaultMessage("CLI.HelpLong"), name),
		RunE: func(c *cobra.Command, args []string) error {
			cmd, _, err := c.Root().Find(args)
			if cmd == nil || err != nil {
				if _, err := fmt.Fprintf(c.OutOrStdout(), i18n.DefaultMessage("CLI.UnknownHelp"), args); err != nil {
					return fmt.Errorf("write unknown help topic: %w", err)
				}
				return c.Root().Usage()
			}
			cmd.InitDefaultHelpFlag()
			return cmd.Help()
		},
	}
}

// addHelpFlag 将特定应用程序的标志添加到指定的 FlagSet
// 对象。
func addHelpFlag(name string, fs *pflag.FlagSet) {
	fs.BoolP(flagHelp, flagHelpShorthand, false, fmt.Sprintf(i18n.DefaultMessage("CLI.HelpFlag"), name))
}

// addHelpCommandFlag 将应用程序特定命令的标志添加到
// 指定的 FlagSet 对象。
func addHelpCommandFlag(usage string, fs *pflag.FlagSet) {
	fs.BoolP(flagHelp, flagHelpShorthand, false, fmt.Sprintf(i18n.DefaultMessage("CLI.CommandHelpFlag"), color.GreenString(strings.Split(usage, " ")[0])))
}
