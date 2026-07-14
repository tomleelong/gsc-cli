package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"google.golang.org/api/searchconsole/v1"
)

var (
	inspectFormat    string
	inspectInputFile string
)

var inspectCmd = &cobra.Command{
	Use:   "inspect [url]",
	Short: "Inspect the index status of a specific URL",
	Long: `Retrieve indexing details, crawl information, canonical URLs, and usability findings
for a specific URL from the Google index using the Search Console URL Inspection API.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if inspectInputFile == "" && len(args) == 0 {
			return fmt.Errorf("accepts 1 arg(s), received 0. Either specify a URL positional argument or use the --file/-i flag for batch inspection")
		}
		if inspectInputFile != "" && len(args) > 0 {
			return fmt.Errorf("cannot specify both a positional URL argument and the --file/-i flag")
		}

		ctx := cmd.Context()
		svc, err := getServices(ctx)
		if err != nil {
			return err
		}

		var urls []string
		if inspectInputFile != "" {
			data, err := os.ReadFile(inspectInputFile)
			if err != nil {
				return fmt.Errorf("failed to read input file: %w", err)
			}
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
					continue
				}
				urls = append(urls, line)
			}
			if len(urls) == 0 {
				return fmt.Errorf("no valid URLs found in input file '%s'", inspectInputFile)
			}
		} else {
			urls = []string{args[0]}
		}

		var hasErrors bool
		for _, url := range urls {
			if inspectFormat == "table" {
				PrintInfo(fmt.Sprintf("Inspecting URL '%s' in property '%s'...", url, siteURL))
			}

			req := &searchconsole.InspectUrlIndexRequest{
				InspectionUrl: url,
				SiteUrl:       siteURL,
			}

			resp, err := svc.SearchConsole.UrlInspection.Index.Inspect(req).Context(ctx).Do()
			if err != nil {
				PrintError(fmt.Sprintf("URL inspection failed for %s: %v", url, err))
				hasErrors = true
				continue
			}

			if resp.InspectionResult == nil {
				PrintError(fmt.Sprintf("No inspection result returned from Google Search Console for %s", url))
				hasErrors = true
				continue
			}

			res := resp.InspectionResult

			switch strings.ToLower(inspectFormat) {
			case "table":
				printInspectionReport(url, siteURL, res)
			case "json":
				jsonData, err := json.MarshalIndent(res, "", "  ")
				if err != nil {
					PrintError(fmt.Sprintf("Failed to marshal JSON for %s: %v", url, err))
					hasErrors = true
					continue
				}
				fmt.Println(string(jsonData))
			default:
				return fmt.Errorf("unsupported format '%s'. Supported: table, json", inspectFormat)
			}
		}

		if inspectInputFile != "" {
			if hasErrors {
				PrintWarning("Batch URL inspection completed with some errors.")
			} else {
				PrintSuccess("Batch URL inspection completed successfully.")
			}
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
	inspectCmd.Flags().StringVarP(&inspectInputFile, "file", "i", "", "Path to a text file containing one URL per line for batch inspection")

	RootCmd.AddCommand(inspectCmd)
}
