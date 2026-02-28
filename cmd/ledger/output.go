package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
)

// PrintTable renders headers and rows as a table to stdout.
func PrintTable(headers []string, rows [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	hdr := make([]any, len(headers))
	for i, h := range headers {
		hdr[i] = h
	}
	table.Header(hdr...)
	for _, row := range rows {
		r := make([]any, len(row))
		for i, v := range row {
			r[i] = v
		}
		_ = table.Append(r...)
	}
	_ = table.Render()
}

// PrintJSON pretty-prints v as JSON to stdout.
func PrintJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// fmtFloat formats a float64 with up to 8 significant decimal places,
// trimming trailing zeros.
func fmtFloat(f float64) string {
	return fmt.Sprintf("%g", f)
}

// fmtFloat2 formats a float64 to 2 decimal places.
func fmtFloat2(f float64) string {
	return fmt.Sprintf("%.2f", f)
}

// fmtQty formats a quantity for table display — avoids scientific notation and
// limits to a readable number of significant digits.
func fmtQty(f float64) string {
	if f == 0 {
		return "0"
	}
	abs := f
	if abs < 0 {
		abs = -f
	}
	// For large quantities use commas-style grouping isn't available, but at
	// least avoid scientific notation.
	if abs >= 1000 {
		// Round to whole number for very large quantities
		return fmt.Sprintf("%.0f", f)
	}
	if abs >= 1 {
		return fmt.Sprintf("%.4g", f)
	}
	// Small quantities: use scientific notation with 4 sig figs
	return fmt.Sprintf("%.4g", f)
}

// fmtPrice formats a price — small prices keep scientific notation but with
// limited precision; larger prices show up to 6 decimal places.
func fmtPrice(f float64) string {
	if f == 0 {
		return "0"
	}
	abs := f
	if abs < 0 {
		abs = -f
	}
	if abs >= 0.01 {
		// Show up to 6 sig figs, trim trailing zeros
		s := fmt.Sprintf("%.6f", f)
		// Trim trailing zeros after decimal point
		for len(s) > 1 && s[len(s)-1] == '0' {
			s = s[:len(s)-1]
		}
		if s[len(s)-1] == '.' {
			s = s[:len(s)-1]
		}
		return s
	}
	// Very small prices: scientific with 3 sig figs
	return fmt.Sprintf("%.3e", f)
}

// fmtFee formats a fee amount — max 4 decimal places, trailing zeros trimmed.
func fmtFee(f float64) string {
	if f == 0 {
		return "0"
	}
	s := fmt.Sprintf("%.4f", f)
	for len(s) > 1 && s[len(s)-1] == '0' {
		s = s[:len(s)-1]
	}
	if s[len(s)-1] == '.' {
		s = s[:len(s)-1]
	}
	return s
}

// fmtTime formats a time.Time as a short human-readable string.
func fmtTime(t time.Time) string {
	return t.UTC().Format("2006-01-02 15:04:05")
}

// fmtTimeShort formats a time.Time without the year component.
func fmtTimeShort(t time.Time) string {
	return t.UTC().Format("01-02 15:04:05")
}
