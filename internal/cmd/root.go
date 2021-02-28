package cmd

import (
	"github.com/spf13/cobra"

	"github.com/IBM/kubectl-odlm/internal/action"
)

func Execute() error {
	if err := newCmd().Execute(); err != nil {
		return err
	}
	return nil
}

func newCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "odlm",
		Short: "Manage operand deployment lifecycle manager resources in a cluster from the command line",
		Long: `Manage operand deployment lifecycle manager resources in a cluster from the command line.
odlm helps you browse Operand Deployment Lifecycle Manager resources from the command line.`,
	}

	flags := cmd.PersistentFlags()

	var cfg action.Configuration
	cfg.BindFlags(flags)

	cmd.PersistentPreRunE = func(*cobra.Command, []string) error {
		return cfg.Load()
	}

	cmd.AddCommand(
		newODLMTreeCmd(&cfg),
		newVersionCmd(),
	)

	return cmd
}
