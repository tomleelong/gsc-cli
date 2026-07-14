package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"google.golang.org/api/searchconsole/v1"
)

var (
	inspectFormat string
)

var inspectCmd = &cobra.Command{
	Use:   "inspect [url]",
	Short: "Inspect the index status of a specific URL",
	Long: `Retrieve indexing details, crawl information, canonical URLs, and usability findings
for a specific URL from the Google index using the Search Console URL Inspection API.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		svc, err := getServices(ctx)
		if err != nil {
			return err
		}

		inspectURL := args[0]

		if inspectFormat == "table" {
			PrintInfo(fmt.Sprintf("Inspecting URL '%s' in property '%s'...", inspectURL, siteURL))
		}

		req := &searchconsole.InspectUrlIndexRequest{
			InspectionUrl: inspectURL,
			SiteUrl:       siteURL,
		}

		resp, err := svc.SearchConsole.UrlInspection.Index.Inspect(req).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("URL inspection failed: %w", err)
		}

		if resp.InspectionResult == nil {
			return fmt.Errorf("no inspection result returned from Google Search Console")
		}

		res := resp.InspectionResult

		switch strings.ToLower(inspectFormat) {
		case "table":
			printInspectionReport(inspectURL, siteURL, res)
		case "json":
			jsonData, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonData))
		default:
			return fmt.Errorf("unsupported format '%s'. Supported: table, json", inspectFormat)
		}

		return nil
	},
}

func printInspectionReport(url string, site string, res *searchconsole.UrlInspectionResult) {
	fmt.Println()
	fmt.Printf("  %s %s\n", BoldCyan("🔍 URL INSPECTION REPORT:"), Bold(url))
	fmt.Println(Gray("  " + strings.Repeat("─", 65)))

	// 1. Overall Status
	statusIdx := res.IndexStatusResult
	var statusText string
	if statusIdx != nil {
		switch statusIdx.Verdict {
		case "PASS":
			statusText = BoldGreen("✔ INDEXED (PASS)")
		case "FAIL":
			statusText = BoldRed("✘ ERROR (FAIL)")
		case "NEUTRAL":
			statusText = BoldYellow("⚠ EXCLUDED (NEUTRAL)")
		default:
			statusText = Gray("ℹ UNKNOWN (UNSPECIFIED)")
		}
	} else {
		statusText = Gray("No indexing status result available")
	}

	fmt.Printf("  %-25s %s\n", Bold("Status Verdict:"), statusText)
	if statusIdx != nil && statusIdx.CoverageState != "" {
		fmt.Printf("  %-25s %s\n", Bold("Coverage State:"), statusIdx.CoverageState)
	}

	// 2. Indexing Details
	if statusIdx != nil {
		fmt.Println()
		fmt.Println(Bold("  [Index Coverage & Crawl details]"))
		fmt.Printf("  %-25s %s\n", "Last Crawl Time:", formatVal(statusIdx.LastCrawlTime))
		fmt.Printf("  %-25s %s\n", "Crawled As (User Agent):", formatVal(statusIdx.CrawledAs))
		fmt.Printf("  %-25s %s\n", "Page Fetch Status:", formatVal(statusIdx.PageFetchState))
		fmt.Printf("  %-25s %s\n", "Indexing Allowed (Meta):", formatVal(statusIdx.IndexingState))
		fmt.Printf("  %-25s %s\n", "Robots.txt Crawl Status:", formatVal(statusIdx.RobotsTxtState))

		// Discovery Info
		if len(statusIdx.Sitemap) > 0 {
			fmt.Printf("  %-25s %s\n", "Sitemaps:", strings.Join(statusIdx.Sitemap, ", "))
		} else {
			fmt.Printf("  %-25s %s\n", "Sitemaps:", Gray("None detected"))
		}

		if len(statusIdx.ReferringUrls) > 0 {
			fmt.Printf("  %-25s %s\n", "Referring URLs:", strings.Join(statusIdx.ReferringUrls, "\n                            "))
		}
	}

	// 3. Canonicalization
	if statusIdx != nil {
		fmt.Println()
		fmt.Println(Bold("  [Canonicalization]"))
		fmt.Printf("  %-25s %s\n", "User Declared Canonical:", formatVal(statusIdx.UserCanonical))
		fmt.Printf("  %-25s %s\n", "Google Selected Canonical:", formatVal(statusIdx.GoogleCanonical))
	}

	// 4. Mobile Usability Result
	fmt.Println()
	fmt.Println(Bold("  [Enhancements & Usability]"))
	if res.MobileUsabilityResult != nil {
		mobVerdict := res.MobileUsabilityResult.Verdict
		var styledMob string
		switch mobVerdict {
		case "PASS":
			styledMob = BoldGreen("Pass")
		case "FAIL":
			styledMob = BoldRed("Fail")
		default:
			styledMob = Gray(mobVerdict)
		}
		fmt.Printf("  %-25s %s\n", "Mobile Usability Verdict:", styledMob)
		if len(res.MobileUsabilityResult.Issues) > 0 {
			for _, issue := range res.MobileUsabilityResult.Issues {
				fmt.Printf("    - %s (%s)\n", BoldRed(issue.Message), issue.Severity)
			}
		}
	} else {
		fmt.Printf("  %-25s %s\n", "Mobile Usability Verdict:", Gray("Not checked / Not applicable"))
	}

	// 5. Rich Results
	if res.RichResultsResult != nil {
		richVerdict := res.RichResultsResult.Verdict
		var styledRich string
		switch richVerdict {
		case "PASS":
			styledRich = BoldGreen("Pass")
		case "FAIL":
			styledRich = BoldRed("Fail")
		default:
			styledRich = Gray(richVerdict)
		}
		fmt.Printf("  %-25s %s\n", "Rich Schema Verdict:", styledRich)

		if len(res.RichResultsResult.DetectedItems) > 0 {
			var schemas []string
			for _, item := range res.RichResultsResult.DetectedItems {
				schemas = append(schemas, fmt.Sprintf("%s (%d item(s))", item.RichResultType, len(item.Items)))
			}
			fmt.Printf("    Schemas Detected:     %s\n", strings.Join(schemas, ", "))
		}
	} else {
		fmt.Printf("  %-25s %s\n", "Rich Schema Verdict:", Gray("None detected"))
	}

	// 6. Online Portal URL
	if res.InspectionResultLink != "" {
		fmt.Println()
		fmt.Printf("  %-25s %s\n", Bold("Web Interface Link:"), Dim(res.InspectionResultLink))
	}
	fmt.Println()
}

func formatVal(val string) string {
	if val == "" || val == "INDEXING_STATE_UNSPECIFIED" || val == "PAGE_FETCH_STATE_UNSPECIFIED" || val == "ROBOTS_TXT_STATE_UNSPECIFIED" {
		return Gray("Unknown / Not available")
	}
	// Return nice highlights for standard pass statuses
	switch val {
	case "SUCCESSFUL", "ALLOWED", "INDEXING_ALLOWED":
		return BoldGreen(val)
	case "BLOCKED_BY_META_TAG", "BLOCKED_BY_HTTP_HEADER", "DISALLOWED", "NOT_FOUND", "SERVER_ERROR", "ACCESS_DENIED":
		return BoldRed(val)
	default:
		return val
	}
}

func init() {
	_ = inspectCmd.MarkFlagRequired("site")

	inspectCmd.Flags().StringVarP(&inspectFormat, "format", "f", "table", "Output format (table, json)")

	RootCmd.AddCommand(inspectCmd)
}
