package main

import (
	"github.com/nimatrueway/unbound-ssh/internal"
	"github.com/nimatrueway/unbound-ssh/internal/config"
	"github.com/nimatrueway/unbound-ssh/internal/view"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/ztrue/tracerr"
	"os"
	"runtime/debug"
)

var RootCmd = &cobra.Command{
	Use:   "unbound-ssh",
	Short: "Port forward and file transfer through limited interactive ssh",
	Long: `This tool enables you to port forward and file transfer when you only have limited interactive ssh access
to your server. unbound-ssh will even work if you need to jump multiple hops to reach to your server, or if you are running tmux on top.`,
}

var RootFlags struct {
	Config string
}

var ListenCmd = &cobra.Command{
	Use:   "listen <ssh command> <...ssh args>",
	Short: "Run in listen mode and tap into the remote session stdin/stdout launched by <ssh command>",
	Long:  ``,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := Listen(args)

		if err != nil {
			tracerr.PrintSourceColor(err)
		}
	},
}

var SpyCmd = &cobra.Command{
	Use:   "spy",
	Short: "Run in spy mode on the remote server and connects to the instance that is running in listen mode on the other side.",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		err := Spy()

		if err != nil {
			view.AppendRaw(tracerr.SprintSource(err))
			panic(err)
		}
	},
}

func init() {
	// shared flags
	for _, fs := range []*pflag.FlagSet{ListenCmd.Flags(), SpyCmd.Flags()} {
		fs.StringVarP(&RootFlags.Config, "config", "c", "config.toml", "config file")
	}
	RootCmd.AddCommand(ListenCmd)
	RootCmd.AddCommand(SpyCmd)
	RootCmd.Version = config.Version
}

func Listen(cmd []string) error {
	config.Mode = "listen"
	if err := configure(RootFlags.Config); err != nil {
		return err
	}

	return internal.Listen(cmd)
}

func Spy() error {
	defer func() {
		// because spy-mode can not dump its error on stdout
		// stdin/stdout is to be used for communication with listen-mode
		if err := recover(); err != nil {
			view.AppendRaw("\n\n --- PANIC --- \n\n")
			view.AppendRaw(err.(error).Error())
			view.AppendRaw("\n\n --- STACK TRACE --- \n\n")
			view.AppendRaw(string(debug.Stack()))
			panic(err)
		}
	}()

	config.Mode = "spy"
	if err := configure(RootFlags.Config); err != nil {
		return err
	}

	return internal.Spy()
}

func configure(file string) error {
	err := (&config.Config).Load(file)
	if err != nil {
		return err
	}

	view.Init()
	return nil
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
