package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gsc-cli/auth"
)

var (
	credsFile        string
	clientSecretFile string
	verbose          bool
	siteURL          string
	gscServices      *auth.Services
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "gsc-cli",
	Short: "gsc-cli is a lightweight, premium command-line tool for Google Search Console",
	Long: `A clean, powerful Command Line Interface built in Go to interact with Google Search Console APIs.
Manage sites, submit sitemaps, query search performance analytics, and inspect URLs directly from your terminal.`,
	Run: func(cmd *cobra.Command, args []string) {
		PrintBanner()
		_ = cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Printf("\n%s %v\n", BoldRed("Error:"), err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&credsFile, "credentials", "c", "", "Path to Google Service Account JSON key file (overrides GOOGLE_APPLICATION_CREDENTIALS)")
	RootCmd.PersistentFlags().StringVarP(&clientSecretFile, "client-secret", "s", "", "Path to Google OAuth2 Client Secret JSON file")
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	RootCmd.PersistentFlags().StringVarP(&siteURL, "site", "u", "", "Verified property URL as listed in GSC (e.g. 'sc-domain:example.com' or 'https://example.com/')")
}

// getServices lazily initializes and returns GSC services.
func getServices(ctx context.Context) (*auth.Services, error) {
	if gscServices == nil {
		services, err := auth.GetServices(ctx, credsFile, clientSecretFile)
		if err != nil {
			return nil, err
		}
		gscServices = services
	}
	return gscServices, nil
}
