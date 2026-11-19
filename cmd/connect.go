package cmd

import (
	"errors"
	"fmt"

	"github.com/oschwand/gristdl/internal/grist"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Test connection to a Grist server and list accessible orgs",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := grist.New(viper.GetString("server"), viper.GetString("token"))
		if err != nil {
			return err
		}

		orgs, err := client.Orgs(cmd.Context())
		var apiErr *grist.APIError
		if err != nil {
			if errors.As(err, &apiErr) && (apiErr.Status == 401 || apiErr.Status == 403) {
				return fmt.Errorf("%w\nCheck your API token: create one in Grist under Profile Settings > API", err)
			}
			return err
		}

		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "Connected to %s\n", client.Server)
		if len(orgs) == 0 {
			fmt.Fprintln(out, "No accessible orgs on this server.")
			return nil
		}
		fmt.Fprintf(out, "%d org(s):\n", len(orgs))
		for _, org := range orgs {
			domain := ""
			if org.Domain != nil {
				domain = *org.Domain
			}
			fmt.Fprintf(out, "  %d\t%s\t%s\t%s\n", org.ID, org.Name, domain, org.Access)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(connectCmd)
}
