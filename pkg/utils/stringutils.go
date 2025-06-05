// pkg/utils/stringutils.go
package utils

import "strings"

// Ellipsis shortens a string to a maximum length, adding "..." if truncated.
func Ellipsis(s string, maxLength int) string {
	s = strings.TrimSpace(s)             // Trim spaces first
	s = strings.ReplaceAll(s, "\n", " ") // Replace newlines with spaces for single line snippet
	s = strings.ReplaceAll(s, "\r", "")

	if len(s) <= maxLength {
		return s
	}
	if maxLength <= 3 { // Not enough space for "..."
		return s[:maxLength]
	}
	return s[:maxLength-3] + "..."
}
