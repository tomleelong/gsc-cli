package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// sitemapsCmd represents the sitemaps command

var sitemapsCmd = &cobra.Command{
	Use:   "sitemaps",
	Short: "Manage sitemaps for a property",
	Long:  "List, submit, or delete sitemaps for a specific Google Search Console property.",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var sitemapsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List submitted sitemaps for a property",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		svc, err := getServices(ctx)
		if err != nil {
			return err
		}

		PrintInfo(fmt.Sprintf("Fetching submitted sitemaps for property '%s'...", siteURL))

		res, err := svc.Webmasters.Sitemaps.List(siteURL).Do()
		if err != nil {
			return fmt.Errorf("failed to list sitemaps: %w", err)
		}

		if len(res.Sitemap) == 0 {
			PrintWarning("No sitemaps found for this property.")
			return nil
		}

		headers := []string{"Sitemap URL", "Type", "Status", "Submitted", "Downloaded", "URLs (Sub/Idx)", "Errors", "Warnings"}
		var rows [][]string

		for _, s := range res.Sitemap {
			// Status
			status := BoldGreen("Processed")
			if s.IsPending {
				status = Yellow("Pending")
			}

			// Dates
			subDate := s.LastSubmitted
			if subDate == "" {
				subDate = "-"
			}
			dlDate := s.LastDownloaded
			if dlDate == "" {
				dlDate = "-"
			}

			// Submitted / Indexed URLs
			stats := []string{}
			for _, content := range s.Contents {
				stats = append(stats, fmt.Sprintf("%s: %d/%d", content.Type, content.Submitted, content.Indexed))
			}
			statsStr := strings.Join(stats, "\n")
			if statsStr == "" {
				statsStr = "-"
			}

			// Errors / Warnings
			errStr := "-"
			if s.Errors > 0 {
				errStr = BoldRed(fmt.Sprintf("%d", s.Errors))
			}
			warnStr := "-"
			if s.Warnings > 0 {
				warnStr = BoldYellow(fmt.Sprintf("%d", s.Warnings))
			}

			rows = append(rows, []string{
				s.Path,
				s.Type,
				status,
				subDate,
				dlDate,
				statsStr,
				errStr,
				warnStr,
			})
		}

		fmt.Println()
		RenderTable(headers, rows)
		PrintSuccess(fmt.Sprintf("Listed %d sitemap(s).", len(res.Sitemap)))
		return nil
	},
}

var sitemapsAddCmd = &cobra.Command{
	Use:   "add [sitemap-url]",
	Short: "Submit a new sitemap for a property",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		svc, err := getServices(ctx)
		if err != nil {
			return err
		}

		sitemapURL := args[0]
		PrintInfo(fmt.Sprintf("Submitting sitemap '%s' to property '%s'...", sitemapURL, siteURL))

		err = svc.Webmasters.Sitemaps.Submit(siteURL, sitemapURL).Do()
		if err != nil {
			return fmt.Errorf("failed to submit sitemap: %w", err)
		}

		PrintSuccess(fmt.Sprintf("Successfully submitted sitemap '%s' to Google Search Console.", sitemapURL))
		return nil
	},
}

var sitemapsDeleteForce bool

var sitemapsDeleteCmd = &cobra.Command{
	Use:   "delete [sitemap-url]",
	Short: "Delete a submitted sitemap from a property",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		svc, err := getServices(ctx)
		if err != nil {
			return err
		}

		sitemapURL := args[0]

		confirmed, err := ConfirmDestructiveAction(
			fmt.Sprintf("You are about to delete sitemap '%s' from property '%s'.", sitemapURL, siteURL),
			sitemapsDeleteForce,
		)
		if err != nil {
			return err
		}
		if !confirmed {
			PrintWarning("Deletion cancelled.")
			return nil
		}

		PrintInfo(fmt.Sprintf("Deleting sitemap '%s' from property '%s'...", sitemapURL, siteURL))

		err = svc.Webmasters.Sitemaps.Delete(siteURL, sitemapURL).Do()
		if err != nil {
			return fmt.Errorf("failed to delete sitemap: %w", err)
		}

		PrintSuccess(fmt.Sprintf("Successfully deleted sitemap '%s' from Google Search Console.", sitemapURL))
		return nil
	},
}

func init() {
	_ = sitemapsCmd.MarkPersistentFlagRequired("site")

	sitemapsDeleteCmd.Flags().BoolVarP(&sitemapsDeleteForce, "force", "y", false, "Force delete without confirmation prompt")

	sitemapsCmd.AddCommand(sitemapsListCmd)
	sitemapsCmd.AddCommand(sitemapsAddCmd)
	sitemapsCmd.AddCommand(sitemapsDeleteCmd)
	RootCmd.AddCommand(sitemapsCmd)
}
