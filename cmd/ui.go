package cmd

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// ANSI color escape codes
const (
	ColorReset   = "\033[0m"
	ColorBold    = "\033[1m"
	ColorDim     = "\033[2m"
	ColorItalic  = "\033[3m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorWhite   = "\033[37m"
	ColorGray    = "\033[90m"

	// Bold variants
	ColorBoldRed     = "\033[1;31m"
	ColorBoldGreen   = "\033[1;32m"
	ColorBoldYellow  = "\033[1;33m"
	ColorBoldBlue    = "\033[1;34m"
	ColorBoldMagenta = "\033[1;35m"
	ColorBoldCyan    = "\033[1;36m"
)

var ansiRegexp = regexp.MustCompile(`\033\[[0-9;]*m`)

// displayLength returns the length of string without ANSI escape sequences.
func displayLength(s string) int {
	clean := ansiRegexp.ReplaceAllString(s, "")
	return len(clean)
}

// Color wrappers
func Bold(s string) string    { return ColorBold + s + ColorReset }
func Dim(s string) string     { return ColorDim + s + ColorReset }
func Red(s string) string     { return ColorRed + s + ColorReset }
func Green(s string) string   { return ColorGreen + s + ColorReset }
func Yellow(s string) string  { return ColorYellow + s + ColorReset }
func Blue(s string) string    { return ColorBlue + s + ColorReset }
func Magenta(s string) string { return ColorMagenta + s + ColorReset }
func Cyan(s string) string    { return ColorCyan + s + ColorReset }
func Gray(s string) string    { return ColorGray + s + ColorReset }

func BoldRed(s string) string    { return ColorBoldRed + s + ColorReset }
func BoldGreen(s string) string  { return ColorBoldGreen + s + ColorReset }
func BoldYellow(s string) string { return ColorBoldYellow + s + ColorReset }
func BoldBlue(s string) string   { return ColorBoldBlue + s + ColorReset }
func BoldCyan(s string) string   { return ColorBoldCyan + s + ColorReset }

// Status alerts
func PrintSuccess(msg string) {
	fmt.Printf("%s %s\n", BoldGreen("✔"), msg)
}

func PrintError(msg string) {
	fmt.Printf("%s %s\n", BoldRed("✖"), msg)
}

func PrintWarning(msg string) {
	fmt.Printf("%s %s\n", BoldYellow("⚠"), msg)
}

func PrintInfo(msg string) {
	fmt.Printf("%s %s\n", BoldCyan("ℹ"), msg)
}

// PrintBanner prints a premium CLI title banner
func PrintBanner() {
	banner := `
  %s%s┌────────────────────────────────────────────────────────┐
  │ %s%-54s%s │
  │ %s%-54s%s │
  └────────────────────────────────────────────────────────┘%s
`
	fmt.Printf(banner,
		ColorBold, ColorCyan,
		ColorBoldMagenta, "GOOGLE SEARCH CONSOLE CLI (gsc-cli)", ColorCyan,
		ColorGray, "A premium command-line tool for SEO performance & index data", ColorCyan,
		ColorReset,
	)
}

// RenderTable displays data in a beautifully aligned, minimalist table format.
func RenderTable(headers []string, rows [][]string) {
	RenderTableToWriter(os.Stdout, headers, rows)
}

// RenderTableToWriter renders the table to a specified io.Writer.
func RenderTableToWriter(w io.Writer, headers []string, rows [][]string) {
	if len(headers) == 0 {
		return
	}

	numCols := len(headers)
	colWidths := make([]int, numCols)

	// Compute initial column widths from headers
	for i, h := range headers {
		colWidths[i] = displayLength(h)
	}

	// Adjust column widths based on row contents
	for _, row := range rows {
		for i := 0; i < numCols && i < len(row); i++ {
			dl := displayLength(row[i])
			if dl > colWidths[i] {
				colWidths[i] = dl
			}
		}
	}

	// Print headers in bold
	var headerLine strings.Builder
	var separatorLine strings.Builder

	headerLine.WriteString("  ")
	separatorLine.WriteString("  ")

	for i, h := range headers {
		dl := displayLength(h)
		padding := colWidths[i] - dl
		headerLine.WriteString(Bold(h))
		headerLine.WriteString(strings.Repeat(" ", padding))

		separatorLine.WriteString(strings.Repeat("─", colWidths[i]))

		// Add space between columns, but not after the last column
		if i < numCols-1 {
			headerLine.WriteString("   ")
			separatorLine.WriteString("   ")
		}
	}
	fmt.Fprintln(w, headerLine.String())
	fmt.Fprintln(w, Gray(separatorLine.String()))

	// Print rows
	for _, row := range rows {
		var rowLine strings.Builder
		rowLine.WriteString("  ")
		for i := 0; i < numCols; i++ {
			val := ""
			if i < len(row) {
				val = row[i]
			}
			dl := displayLength(val)
			padding := colWidths[i] - dl
			rowLine.WriteString(val)
			rowLine.WriteString(strings.Repeat(" ", padding))

			if i < numCols-1 {
				rowLine.WriteString("   ")
			}
		}
		fmt.Fprintln(w, rowLine.String())
	}
	fmt.Fprintln(w)
}
