package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/api/webmasters/v3"
)

var (
	queryStartDate  string
	queryEndDate    string
	queryDimensions []string
	queryFilters    []string
	queryRowLimit   int64
	queryFormat     string
	queryOutputFile string
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query search analytics performance data",
	Long: `Query impressions, clicks, CTR, and average position from Google Search Console.
Filter by search query, page URL, country, device, or search appearance, and group results by multiple dimensions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		svc, err := getServices(ctx)
		if err != nil {
			return err
		}

		// Calculate default dates if not provided
		if queryEndDate == "" {
			// Search Console is typically delayed by 2 days
			queryEndDate = time.Now().AddDate(0, 0, -2).Format("2006-01-02")
		}
		if queryStartDate == "" {
			// Default to 30 days before the end date
			end, err := time.Parse("2006-01-02", queryEndDate)
			if err != nil {
				return fmt.Errorf("invalid end date format: %w", err)
			}
			queryStartDate = end.AddDate(0, 0, -30).Format("2006-01-02")
		}

		// Prepare request
		req := &webmasters.SearchAnalyticsQueryRequest{
			StartDate: queryStartDate,
			EndDate:   queryEndDate,
			RowLimit:  queryRowLimit,
		}

		// Map dimensions
		if len(queryDimensions) > 0 {
			for _, d := range queryDimensions {
				dim := strings.TrimSpace(d)
				switch strings.ToLower(dim) {
				case "query", "page", "country", "device", "date", "searchappearance", "search_appearance":
					if strings.ToLower(dim) == "search_appearance" {
						dim = "searchAppearance"
					}
					req.Dimensions = append(req.Dimensions, dim)
				default:
					return fmt.Errorf("unsupported dimension '%s'. Supported dimensions: query, page, country, device, date, searchAppearance", dim)
				}
			}
		} else {
			// Default dimension
			req.Dimensions = []string{"query"}
		}

		// Map filters
		if len(queryFilters) > 0 {
			var filters []*webmasters.ApiDimensionFilter

			for _, f := range queryFilters {
				parts := strings.SplitN(strings.TrimSpace(f), " ", 3)
				if len(parts) < 3 {
					return fmt.Errorf("invalid filter format '%s'. Must be: '<dimension> <operator> <expression>' (e.g. 'device == MOBILE' or 'query contains seo')", f)
				}

				dim := strings.ToLower(strings.TrimSpace(parts[0]))
				op := strings.TrimSpace(parts[1])
				expr := strings.Trim(strings.TrimSpace(parts[2]), "\"'")

				// Format dimension name
				switch dim {
				case "query", "page", "country", "device", "searchappearance", "search_appearance":
					if dim == "search_appearance" {
						dim = "searchAppearance"
					}
				default:
					return fmt.Errorf("unsupported filter dimension '%s' in '%s'. Supported: query, page, country, device, searchAppearance", dim, f)
				}

				// Map operator
				var apiOp string
				switch strings.ToLower(op) {
				case "contains":
					apiOp = "contains"
				case "equals", "==":
					apiOp = "equals"
				case "notcontains", "not_contains":
					apiOp = "notContains"
				case "notequals", "not_equals", "!=":
					apiOp = "notEquals"
				case "includingregex", "regex", "~":
					apiOp = "includingRegex"
				case "excludingregex", "!~":
					apiOp = "excludingRegex"
				default:
					return fmt.Errorf("unsupported filter operator '%s' in '%s'. Supported: contains, equals (==), notContains, notEquals (!=), includingRegex (~), excludingRegex (!~)", op, f)
				}

				filters = append(filters, &webmasters.ApiDimensionFilter{
					Dimension:  dim,
					Operator:   apiOp,
					Expression: expr,
				})
			}

			// Add filters to query request
			req.DimensionFilterGroups = []*webmasters.ApiDimensionFilterGroup{
				{
					GroupType: "and",
					Filters:   filters,
				},
			}
		}

		// Run request
		if queryFormat == "table" {
			PrintInfo(fmt.Sprintf("Querying Search Analytics for property '%s'...", siteURL))
			PrintInfo(fmt.Sprintf("Date Range: %s to %s | Dimensions: %s | Limit: %d", queryStartDate, queryEndDate, strings.Join(req.Dimensions, ", "), queryRowLimit))
			if len(queryFilters) > 0 {
				PrintInfo(fmt.Sprintf("Filters applied: %s", strings.Join(queryFilters, " AND ")))
			}
		}

		resp, err := svc.Webmasters.Searchanalytics.Query(siteURL, req).Do()
		if err != nil {
			return fmt.Errorf("analytics query failed: %w", err)
		}

		if len(resp.Rows) == 0 {
			if queryOutputFile != "" {
				if strings.ToLower(queryFormat) == "json" {
					if err := os.WriteFile(queryOutputFile, []byte("[]\n"), 0644); err != nil {
						return fmt.Errorf("failed to write empty json to output file: %w", err)
					}
					PrintSuccess(fmt.Sprintf("Successfully wrote empty JSON output to %s", queryOutputFile))
				} else if strings.ToLower(queryFormat) == "csv" {
					f, err := os.Create(queryOutputFile)
					if err != nil {
						return fmt.Errorf("failed to create output file: %w", err)
					}
					defer f.Close()
					writer := csv.NewWriter(f)
					headers := []string{}
					for _, d := range req.Dimensions {
						headers = append(headers, d)
					}
					headers = append(headers, "clicks", "impressions", "ctr", "position")
					_ = writer.Write(headers)
					writer.Flush()
					PrintSuccess(fmt.Sprintf("Successfully wrote CSV headers to %s", queryOutputFile))
				} else { // table
					if err := os.WriteFile(queryOutputFile, []byte("No search performance data found matching your query criteria.\n"), 0644); err != nil {
						return fmt.Errorf("failed to write empty table to output file: %w", err)
					}
					PrintSuccess(fmt.Sprintf("Successfully wrote empty results info to %s", queryOutputFile))
				}
			} else {
				if queryFormat == "table" {
					PrintWarning("No search performance data found matching your query criteria.")
				} else if queryFormat == "json" {
					_, _ = fmt.Fprint(os.Stdout, "[]\n")
				}
			}
			return nil
		}

		// Output results based on format
		switch strings.ToLower(queryFormat) {
		case "table":
			headers := []string{}
			// Set dimension headers
			for _, d := range req.Dimensions {
				headers = append(headers, strings.ToUpper(d))
			}
			headers = append(headers, "CLICKS", "IMPRESSIONS", "CTR", "POSITION")

			var rows [][]string
			for _, row := range resp.Rows {
				rowCells := []string{}
				// Append dimension keys
				for i := range req.Dimensions {
					val := ""
					if i < len(row.Keys) {
						val = row.Keys[i]
					}
					rowCells = append(rowCells, val)
				}
				// Format metrics
				rowCells = append(rowCells,
					fmt.Sprintf("%.0f", row.Clicks),
					fmt.Sprintf("%.0f", row.Impressions),
					fmt.Sprintf("%.2f%%", row.Ctr*100.0),
					fmt.Sprintf("%.1f", row.Position),
				)
				rows = append(rows, rowCells)
			}

			var tableBuf strings.Builder
			RenderTableToWriter(&tableBuf, headers, rows)
			tableStr := tableBuf.String()

			if queryOutputFile != "" {
				// Strip ANSI codes
				cleanTable := ansiRegexp.ReplaceAllString(tableStr, "")
				err := os.WriteFile(queryOutputFile, []byte(cleanTable), 0644)
				if err != nil {
					return fmt.Errorf("failed to write table to output file: %w", err)
				}
				PrintSuccess(fmt.Sprintf("Successfully wrote table output to %s", queryOutputFile))
			} else {
				fmt.Println()
				fmt.Print(tableStr)
				PrintSuccess(fmt.Sprintf("Retrieved %d rows.", len(resp.Rows)))
			}

		case "csv":
			var writer *csv.Writer
			var f *os.File
			if queryOutputFile != "" {
				var err error
				f, err = os.Create(queryOutputFile)
				if err != nil {
					return fmt.Errorf("failed to create output file: %w", err)
				}
				defer f.Close()
				writer = csv.NewWriter(f)
			} else {
				writer = csv.NewWriter(os.Stdout)
			}
			defer writer.Flush()

			// Headers
			headers := []string{}
			for _, d := range req.Dimensions {
				headers = append(headers, d)
			}
			headers = append(headers, "clicks", "impressions", "ctr", "position")
			if err := writer.Write(headers); err != nil {
				return err
			}

			// Rows
			for _, row := range resp.Rows {
				rowCells := []string{}
				for i := range req.Dimensions {
					val := ""
					if i < len(row.Keys) {
						val = row.Keys[i]
					}
					rowCells = append(rowCells, val)
				}
				rowCells = append(rowCells,
					fmt.Sprintf("%.0f", row.Clicks),
					fmt.Sprintf("%.0f", row.Impressions),
					fmt.Sprintf("%.6f", row.Ctr),
					fmt.Sprintf("%.2f", row.Position),
				)
				if err := writer.Write(rowCells); err != nil {
					return err
				}
			}

			if queryOutputFile != "" {
				PrintSuccess(fmt.Sprintf("Successfully wrote CSV output to %s", queryOutputFile))
			}

		case "json":
			type jsonRow struct {
				Dimensions map[string]string `json:"dimensions"`
				Clicks     float64           `json:"clicks"`
				Impressions float64           `json:"impressions"`
				Ctr        float64           `json:"ctr"`
				Position   float64           `json:"position"`
			}

			outputRows := make([]jsonRow, len(resp.Rows))
			for idx, r := range resp.Rows {
				dims := make(map[string]string)
				for i, d := range req.Dimensions {
					val := ""
					if i < len(r.Keys) {
						val = r.Keys[i]
					}
					dims[d] = val
				}
				outputRows[idx] = jsonRow{
					Dimensions:  dims,
					Clicks:      r.Clicks,
					Impressions: r.Impressions,
					Ctr:         r.Ctr,
					Position:    r.Position,
				}
			}

			jsonData, err := json.MarshalIndent(outputRows, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}

			if queryOutputFile != "" {
				err = os.WriteFile(queryOutputFile, jsonData, 0644)
				if err != nil {
					return fmt.Errorf("failed to write JSON to output file: %w", err)
				}
				PrintSuccess(fmt.Sprintf("Successfully wrote JSON output to %s", queryOutputFile))
			} else {
				fmt.Println(string(jsonData))
			}

		default:
			return fmt.Errorf("unsupported output format '%s'. Supported: table, csv, json", queryFormat)
		}

		return nil
	},
}

func init() {
	_ = queryCmd.MarkFlagRequired("site")

	queryCmd.Flags().StringVarP(&queryStartDate, "start-date", "d", "", "Start date in YYYY-MM-DD format (defaults to 30 days before end date)")
	queryCmd.Flags().StringVarP(&queryEndDate, "end-date", "e", "", "End date in YYYY-MM-DD format (defaults to 2 days ago, standard GSC data delay)")
	queryCmd.Flags().StringSliceVar(&queryDimensions, "dimensions", []string{"query"}, "Comma-separated list of dimensions to group by (query, page, country, device, date, searchAppearance)")
	queryCmd.Flags().StringSliceVar(&queryFilters, "filter", []string{}, "Filter results by '<dimension> <operator> <value>' (can be specified multiple times). Operators: contains, equals (==), notContains, notEquals (!=), includingRegex (~), excludingRegex (!~)")
	queryCmd.Flags().Int64Var(&queryRowLimit, "limit", 100, "Maximum number of rows to retrieve")
	queryCmd.Flags().StringVarP(&queryFormat, "format", "f", "table", "Output format (table, csv, json)")
	queryCmd.Flags().StringVarP(&queryOutputFile, "output-file", "o", "", "Path to local file where query results will be saved")

	RootCmd.AddCommand(queryCmd)
}
