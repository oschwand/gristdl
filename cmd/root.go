package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/oschwand/gristdl/internal/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	envFile string
)

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

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .gristdl.yaml or $HOME/.gristdl.yaml)")
	rootCmd.PersistentFlags().StringVar(&envFile, "env-file", ".env", "dotenv file to load (set to \"\" to skip; real environment variables take precedence)")
	rootCmd.PersistentFlags().String("server", "https://docs.getgrist.com", "Grist API server")
	rootCmd.PersistentFlags().String("token", "", "Grist API token (or GRISTDL_TOKEN)")

	_ = viper.BindPFlag("env-file", rootCmd.PersistentFlags().Lookup("env-file"))
	_ = viper.BindPFlag("server", rootCmd.PersistentFlags().Lookup("server"))
	_ = viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token"))
	_ = viper.BindEnv("server")
	_ = viper.BindEnv("token")
}

func initConfig() {
	viper.SetEnvPrefix("GRISTDL")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	loadEnvFile(viper.GetString("env-file"))

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".gristdl")
	}

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
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
