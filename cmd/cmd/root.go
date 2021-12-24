package cmd

import (
	"fmt"
	"os"

	"github.com/networkop/declarative-netbox/netbox"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap/zapcore"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var debug bool

func addGlobalFlags(fs *pflag.FlagSet) {
	fs.BoolVarP(&debug, "debug", "d", debug, "Enable debug-level logging")
}

func Execute(version, gitCommit string) error {

	log.SetOutput(os.Stdout)

	logOpts := &zap.Options{
		Development: true,
		Level:       zapcore.InfoLevel,
	}

	logr := zap.New(zap.UseFlagOptions(logOpts))

	cliOpts := []CliOption{
		WithLogger(logr),
	}

	var cli *Cli
	cli, err := NewCli(cliOpts...)
	if err != nil {
		logrus.Error("Error initializing CLI: %s", err)
		os.Exit(1)
	}

	root := &cobra.Command{
		Use:   "nbctl [command]",
		Short: "Unofficial CLI client for netbox",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if debug {
				log.SetLevel(logrus.DebugLevel)
			}

		},
		Version: fmt.Sprintf("version: %q, commit: %q", version, gitCommit),
	}

	cobra.EnableCommandSorting = false
	addGlobalFlags(root.PersistentFlags())

	root.AddCommand(
		NewGetCommand(cli),
		NewApplyCommand(cli),
		NewDeleteCommand(cli),
		NewAuthCommand(cli),
	)

	return root.Execute()
}

func NewAuthCommand(c *Cli) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "login <server> <token>",
		Short: "Login the Netbox server",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {

			auth := NewAuthData()
			if err := auth.SaveAuth(args[0], args[1]); err != nil {
				return err
			}

			if err := netbox.AuthCheck(args[0], args[1]); err != nil {
				return err
			}

			log.Info("Authentication successful")
			return nil
		},
	}
	return cmd
}

func NewGetCommand(cli *Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get",
		Aliases: []string{"read"},
		Short:   "Get one or many resources",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	for _, r := range GetResources(cli) {
		if r.Get() != nil {
			cmd.AddCommand(r.Get())

		}
	}

	return cmd
}

func NewApplyCommand(cli *Cli) *cobra.Command {
	var fileName string
	cmd := &cobra.Command{
		Use:   "apply -f FILENAME",
		Short: "Apply a configuration to a resource by filename",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Print(fileName)
			if fileName == "" {
				cmd.Help()
				return nil
			}
			if err := Action(cli, ApplyAction, fileName); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(&fileName, "filename", "f", "", "filename to apply")
	return cmd
}

func NewDeleteCommand(cli *Cli) *cobra.Command {
	var fileName string
	cmd := &cobra.Command{
		Use:   "delete -f FILENAME",
		Short: "Delete a configuration by filename",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Print(fileName)
			if fileName == "" {
				cmd.Help()
				return nil
			}
			if err := Action(cli, DeleteAction, fileName); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(&fileName, "filename", "f", "", "filename to apply")
	return cmd
}
