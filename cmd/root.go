package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/oschwand/gristdl/internal/grist"
	"github.com/oschwand/gristdl/internal/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var envFile string

var rootCmd = &cobra.Command{
	Use:     "gristdl",
	Short:   "Download Grist documents via the Grist API",
	Version: version.String(),
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&envFile, "env-file", defaultEnvFile(),
		"dotenv config file to load (set to \"\" to skip; real environment variables take precedence)")
	rootCmd.PersistentFlags().String("server", "https://docs.getgrist.com", "Grist API server (or GRIST_URL)")
	rootCmd.PersistentFlags().String("token", "", "Grist API token (or GRIST_API_KEY)")
	rootCmd.PersistentFlags().String("doc-id", "", "default document ID (or GRIST_DOC_ID)")
	rootCmd.PersistentFlags().String("team-id", "", "default team or org domain (or GRIST_TEAM_ID)")

	_ = viper.BindPFlag("env-file", rootCmd.PersistentFlags().Lookup("env-file"))
	_ = viper.BindPFlag("server", rootCmd.PersistentFlags().Lookup("server"))
	_ = viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token"))
	_ = viper.BindPFlag("doc-id", rootCmd.PersistentFlags().Lookup("doc-id"))
	_ = viper.BindPFlag("team-id", rootCmd.PersistentFlags().Lookup("team-id"))

	_ = viper.BindEnv("server", "GRIST_URL")
	_ = viper.BindEnv("token", "GRIST_API_KEY")
	_ = viper.BindEnv("doc-id", "GRIST_DOC_ID")
	_ = viper.BindEnv("team-id", "GRIST_TEAM_ID")
	_ = viper.BindEnv("env-file", "GRIST_ENV_FILE")
}

func initConfig() {
	loadEnvFile(viper.GetString("env-file"))
}

func defaultEnvFile() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "gristdl", "config.env")
}

func loadEnvFile(path string) {
	if path == "" {
		return
	}
	if err := godotenv.Load(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		fmt.Fprintf(os.Stderr, "warning: could not load env file %s: %v\n", path, err)
	}
}

func newGristClient() (*grist.Client, error) {
	return grist.New(viper.GetString("server"), viper.GetString("token"))
}
