// Copyright 2025 Pentora Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");

package format

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// Summary represents operation results for consistent formatting
type Summary struct {
	Operation   string        // Operation name: "install", "update", "uninstall", etc.
	Success     int           // Successful operation count
	Skipped     int           // Skipped operation count
	Failed      int           // Failed operation count
	Errors      []ErrorDetail // First N errors (truncated for display)
	TotalErrors int           // Total error count (for truncation message)
}

// ErrorDetail represents a single error with context
type ErrorDetail struct {
	PluginID  string // Plugin identifier
	Error     string // Error message
	ErrorCode string // Error code for suggestion mapping
}

const (
	maxErrorsToShow = 5 // Maximum errors to display before truncating
)

// PrintSuccessSummary prints a standardized success message
// Examples:
//   - "✓ Installed ssh-weak-cipher v1.0.0"
//   - "✓ Uninstalled 3 plugins"
func (f *formatter) PrintSuccessSummary(operation, pluginID, version string) error {
	if f.quiet {
		// Quiet mode: minimal output
		if pluginID != "" && version != "" {
			_, err := fmt.Fprintf(f.stdout, "%s v%s\n", pluginID, version)
			return err
		}
		return nil
	}

	if f.mode == ModeJSON {
		// JSON mode: structured output
		return f.PrintJSON(map[string]any{
			"success":   true,
			"operation": operation,
			"plugin_id": pluginID,
			"version":   version,
		})
	}

	// Table mode: user-friendly message
	var message string
	if pluginID != "" && version != "" {
		message = fmt.Sprintf("✓ %s %s v%s", capitalize(operation), pluginID, version)
	} else {
		message = fmt.Sprintf("✓ %s completed successfully", capitalize(operation))
	}

	if f.color {
		_, err := color.New(color.FgGreen).Fprintln(f.stdout, message)
		return err
	}

	_, err := fmt.Fprintln(f.stdout, message)
	return err
}

// PrintPartialFailureSummary prints partial failure with counts, errors, and suggestions
// Example output:
//
//	Summary:
//	  ✓ Updated: 3 plugins
//	  ⚠ Skipped: 1 plugin
//	  ✗ Failed:  2 plugins
//
//	Failed plugins:
//	  - ssh-weak-cipher: Remote repository unavailable
//	  - ssh-cve-2024-6387: Checksum mismatch
//
//	💡 Suggestions:
//	  → Retry with different source:  pentora plugin update --source github
//	  → Force re-download:            pentora plugin update --force
func (f *formatter) PrintPartialFailureSummary(summary Summary) error {
	if f.quiet {
		// Quiet mode: suppress summary
		return nil
	}

	if f.mode == ModeJSON {
		// JSON mode: structured output
		return f.PrintJSON(map[string]any{
			"success":       false,
			"partial":       true,
			"operation":     summary.Operation,
			"success_count": summary.Success,
			"skipped_count": summary.Skipped,
			"failed_count":  summary.Failed,
			"errors":        summary.Errors,
		})
	}

	// Table mode: formatted summary
	var sb strings.Builder

	// Print counts
	sb.WriteString("\nSummary:\n")
	if summary.Success > 0 {
		if f.color {
			sb.WriteString(color.GreenString("  ✓ %s: %d\n", capitalize(getCountLabel(summary.Operation, summary.Success)), summary.Success))
		} else {
			sb.WriteString(fmt.Sprintf("  ✓ %s: %d\n", capitalize(getCountLabel(summary.Operation, summary.Success)), summary.Success))
		}
	}
	if summary.Skipped > 0 {
		if f.color {
			sb.WriteString(color.YellowString("  ⚠ Skipped: %d\n", summary.Skipped))
		} else {
			sb.WriteString(fmt.Sprintf("  ⚠ Skipped: %d\n", summary.Skipped))
		}
	}
	if summary.Failed > 0 {
		if f.color {
			sb.WriteString(color.RedString("  ✗ Failed:  %d\n", summary.Failed))
		} else {
			sb.WriteString(fmt.Sprintf("  ✗ Failed:  %d\n", summary.Failed))
		}
	}

	// Print first N errors
	if len(summary.Errors) > 0 {
		sb.WriteString("\nFailed plugins:\n")
		for i, err := range summary.Errors {
			if i >= maxErrorsToShow {
				remaining := summary.TotalErrors - maxErrorsToShow
				sb.WriteString(fmt.Sprintf("  ... and %d more (use --output json for full list)\n", remaining))
				break
			}
			sb.WriteString(fmt.Sprintf("  - %s: %s\n", err.PluginID, err.Error))
		}

		// Print suggestions
		suggestions := collectSuggestions(summary.Errors, summary.Operation)
		if len(suggestions) > 0 {
			sb.WriteString("\n💡 Suggestions:\n")
			for _, s := range suggestions {
				sb.WriteString(fmt.Sprintf("  → %s\n", s))
			}
		}
	}

	_, err := f.stdout.Write([]byte(sb.String()))
	return err
}

// PrintTotalFailureSummary prints total failure with error and suggestions
// Example output:
//
//	✗ Failed to install ssh-weak-cipher: Plugin not found
//
//	💡 Suggestions:
//	  → List available plugins:  pentora plugin list
//	  → Try GitHub source:       pentora plugin install ssh-weak-cipher --source github
func (f *formatter) PrintTotalFailureSummary(operation string, err error, errorCode string) error {
	if f.quiet {
		// Quiet mode: suppress summary
		return nil
	}

	if f.mode == ModeJSON {
		// JSON mode: structured output
		return f.PrintJSON(map[string]any{
			"success":    false,
			"operation":  operation,
			"error":      err.Error(),
			"error_code": errorCode,
		})
	}

	// Table mode: formatted error with suggestions
	var sb strings.Builder

	// Error message
	errorMsg := fmt.Sprintf("✗ Failed to %s: %v", operation, err)
	if f.color {
		sb.WriteString(color.RedString("%s\n", errorMsg))
	} else {
		sb.WriteString(fmt.Sprintf("%s\n", errorMsg))
	}

	// Suggestions based on error code
	suggestions := GetSuggestions(errorCode, operation)
	if len(suggestions) > 0 {
		sb.WriteString("\n💡 Suggestions:\n")
		for _, s := range suggestions {
			sb.WriteString(fmt.Sprintf("  → %s\n", s))
		}
	}

	_, writeErr := f.stdout.Write([]byte(sb.String()))
	return writeErr
}

// GetSuggestions returns actionable hints based on error code and operation
func GetSuggestions(errorCode string, operation string) []string {
	suggestions := []string{}

	switch errorCode {
	case "PLUGIN_NOT_FOUND":
		suggestions = append(suggestions,
			"List available plugins:  pentora plugin list",
			fmt.Sprintf("Try GitHub source:       pentora plugin %s <plugin> --source github", operation),
		)

	case "SERVICE_UNAVAILABLE", "REMOTE_UNAVAILABLE":
		suggestions = append(suggestions,
			fmt.Sprintf("Retry with GitHub:       pentora plugin %s --source github", operation),
			"Check network connection",
		)

	case "CHECKSUM_MISMATCH":
		suggestions = append(suggestions,
			fmt.Sprintf("Force re-download:       pentora plugin %s --force", operation),
		)

	case "VERSION_CONFLICT":
		suggestions = append(suggestions,
			"Uninstall first:         pentora plugin uninstall <plugin>",
			fmt.Sprintf("Force reinstall:         pentora plugin %s --force", operation),
		)

	case "PARTIAL_FAILURE":
		suggestions = append(suggestions,
			fmt.Sprintf("See full details:        pentora plugin %s --output json", operation),
		)

	case "NO_PLUGINS_FOUND":
		suggestions = append(suggestions,
			"Check network connection",
			"Verify DNS resolution",
			fmt.Sprintf("Try GitHub source:       pentora plugin %s <name> --source github", operation),
			fmt.Sprintf("Force re-download:       pentora plugin %s --force", operation),
			"List cached plugins:     pentora plugin list",
		)

	case "INVALID_CATEGORY":
		suggestions = append(suggestions,
			"Valid categories: ssh, http, tls, database, network, web, iot, misc",
			"List plugins:     pentora plugin list",
		)

	case "INVALID_SOURCE":
		suggestions = append(suggestions,
			"Valid sources: official, github",
		)

	case "PLUGIN_NOT_INSTALLED":
		suggestions = append(suggestions,
			"List installed:   pentora plugin list",
			"Install plugin:   pentora plugin install <plugin>",
		)
	}

	return suggestions
}

// collectSuggestions gathers unique suggestions from multiple errors
func collectSuggestions(errors []ErrorDetail, operation string) []string {
	seen := make(map[string]bool)
	var suggestions []string

	for _, err := range errors {
		hints := GetSuggestions(err.ErrorCode, operation)
		for _, hint := range hints {
			if !seen[hint] {
				seen[hint] = true
				suggestions = append(suggestions, hint)
			}
		}
	}

	return suggestions
}

// capitalize capitalizes the first letter of a string
func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// getCountLabel returns the appropriate label for count display
// Examples: "Updated 3" -> "Updated", "Installed 1" -> "Installed"
func getCountLabel(operation string, count int) string {
	switch operation {
	case "install":
		return "installed"
	case "update":
		return "updated"
	case "uninstall":
		return "removed"
	case "clean":
		return "removed"
	case "verify":
		return "verified"
	default:
		return operation
	}
}
