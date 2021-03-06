package cmd

import (
	"fmt"
	"os"

	"github.com/IBM/kubectl-odlm/internal/action"
	"github.com/fatih/color"
	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"

	odlmv1alpha1 "github.com/IBM/operand-deployment-lifecycle-manager/api/v1alpha1"
)

func newODLMTreeCmd(cfg *action.Configuration) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "tree <OperandRequest NAME>",
		SilenceUsage: true, // for when RunE returns an error
		Short:        "Show operator and operand generated by OperandRequest",
		Args:         cobra.MaximumNArgs(1),
		Run: func(command *cobra.Command, args []string) {
			tree := action.NewTree(cfg)
			tree.Ctx = command.Context()
			tree.Table = uitable.New()
			tree.Table.Separator = "  "
			tree.Table.AddRow("NAMESPACE", "NAME", "READY/REASON", "AGE")
			if len(args) == 1 {
				opreqName := args[0]
				tree.TreeView(opreqName)
				fmt.Fprintln(color.Output, tree.Table)
			} else {
				opreqList := &odlmv1alpha1.OperandRequestList{}
				if err := tree.Config.Client.List(tree.Ctx, opreqList, &client.ListOptions{Namespace: tree.Config.Namespace}); err != nil {
					fmt.Println("Error: ", err)
					os.Exit(1)
				}
				for _, opreq := range opreqList.Items {
					tree.TreeView(opreq.Name)
				}
				fmt.Fprintln(color.Output, tree.Table)
			}

		},
	}
	return cmd
}
