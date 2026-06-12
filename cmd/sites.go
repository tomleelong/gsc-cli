package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var sitesCmd = &cobra.Command{
	Use:   "sites",
	Short: "Manage Search Console properties (sites)",
	Long:  "List, add, or delete verified properties (sites) in Google Search Console.",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var sitesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all verified sites in Google Search Console",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		svc, err := getServices(ctx)
		if err != nil {
			return err
		}

		PrintInfo("Fetching verified sites from Google Search Console...")

		res, err := svc.Webmasters.Sites.List().Do()
		if err != nil {
			return fmt.Errorf("failed to list sites: %w", err)
		}

		if len(res.SiteEntry) == 0 {
			PrintWarning("No properties found in your Search Console account.")
			return nil
		}

		headers := []string{"Site URL", "Permission Level"}
		var rows [][]string

		for _, site := range res.SiteEntry {
			permission := site.PermissionLevel
			// Style permission nicely
			var styledPerm string
			switch permission {
			case "siteOwner":
				styledPerm = BoldGreen("Owner (siteOwner)")
			case "siteFullUser":
				styledPerm = BoldCyan("Full User (siteFullUser)")
			case "siteRestrictedUser":
				styledPerm = Yellow("Restricted User (siteRestrictedUser)")
			case "siteUnverifiedUser":
				styledPerm = Gray("Unverified User (siteUnverifiedUser)")
			default:
				styledPerm = permission
			}
			rows = append(rows, []string{site.SiteUrl, styledPerm})
		}

		fmt.Println()
		RenderTable(headers, rows)
		PrintSuccess(fmt.Sprintf("Found %d verified site(s).", len(res.SiteEntry)))
		return nil
	},
}

var sitesAddCmd = &cobra.Command{
	Use:   "add [site-url]",
	Short: "Add a new site to Google Search Console",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		svc, err := getServices(ctx)
		if err != nil {
			return err
		}

		siteURL := args[0]
		PrintInfo(fmt.Sprintf("Adding site '%s' to Google Search Console...", siteURL))

		// Add site
		err = svc.Webmasters.Sites.Add(siteURL).Do()
		if err != nil {
			return fmt.Errorf("failed to add site: %w", err)
		}

		PrintSuccess(fmt.Sprintf("Successfully added site '%s'. Note that you will still need to verify ownership.", siteURL))
		return nil
	},
}

var sitesDeleteCmd = &cobra.Command{
	Use:   "delete [site-url]",
	Short: "Delete an existing site from Google Search Console",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		svc, err := getServices(ctx)
		if err != nil {
			return err
		}

		siteURL := args[0]
		PrintInfo(fmt.Sprintf("Deleting site '%s' from Google Search Console...", siteURL))

		// Delete site
		err = svc.Webmasters.Sites.Delete(siteURL).Do()
		if err != nil {
			return fmt.Errorf("failed to delete site: %w", err)
		}

		PrintSuccess(fmt.Sprintf("Successfully deleted site '%s'.", siteURL))
		return nil
	},
}

func init() {
	sitesCmd.AddCommand(sitesListCmd)
	sitesCmd.AddCommand(sitesAddCmd)
	sitesCmd.AddCommand(sitesDeleteCmd)
	RootCmd.AddCommand(sitesCmd)
}
