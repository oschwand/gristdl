package cmd

import (
	"fmt"

	"github.com/oschwand/gristdl/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version and build information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(cmd.OutOrStdout(), "gristdl version "+version.String())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
