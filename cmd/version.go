package cmd

import (
	"fmt"

	"github.com/oschwand/gristdl/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version, commit and build date",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("gristdl " + version.Full())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
