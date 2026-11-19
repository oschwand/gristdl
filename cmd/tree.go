package cmd

import (
	"fmt"

	"github.com/oschwand/gristdl/internal/grist"
	"github.com/spf13/cobra"
)

var treeCmd = &cobra.Command{
	Use:          "tree",
	Short:        "Display all user documents as a tree, grouped by org and workspace",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newGristClient()
		if err != nil {
			return err
		}

		orgs, err := client.Orgs(cmd.Context())
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()
		total := 0
		for _, org := range orgs {
			wsList, err := client.Workspaces(cmd.Context(), org.ID)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: skipping org %d: %v\n", org.ID, err)
				continue
			}
			var withDocs []grist.Workspace
			for _, ws := range wsList {
				if len(ws.Docs) > 0 {
					withDocs = append(withDocs, ws)
				}
			}
			if len(withDocs) == 0 {
				continue
			}
			if total > 0 {
				fmt.Fprintln(out)
			}
			fmt.Fprintf(out, "%s (%s)\n", org.Name, orgLabel(org))
			for i, ws := range withDocs {
				wsBranch, wsPrefix := branch(i == len(withDocs)-1)
				fmt.Fprintf(out, "%s📁 %s\n", wsBranch, ws.Name)
				for j, doc := range ws.Docs {
					docBranch, _ := branch(j == len(ws.Docs)-1)
					fmt.Fprintf(out, "%s%s📄 %s  %s\n", wsPrefix, docBranch, doc.ID, doc.Name)
				}
			}
			for _, ws := range withDocs {
				total += len(ws.Docs)
			}
		}
		if total == 0 {
			fmt.Fprintln(out, "No documents found.")
		}
		return nil
	},
}

func branch(last bool) (string, string) {
	if last {
		return "└──", "    "
	}
	return "├──", "│   "
}

func orgLabel(org grist.Org) string {
	if org.Domain != nil && *org.Domain != "" {
		return *org.Domain
	}
	return fmt.Sprintf("id %d", org.ID)
}

func init() {
	rootCmd.AddCommand(treeCmd)
}
