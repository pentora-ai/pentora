// Package stringutil provides utility functions for string manipulation.
// Ellipsis shortens a string to a specified maximum length, appending "..." if truncation occurs.
// The function trims leading and trailing spaces from the input string, replaces all newline
// characters with spaces, and removes carriage returns. If the resulting string exceeds maxLength,
// it is truncated and an ellipsis ("...") is appended. If maxLength is less than or equal to 3,
// the function returns the string truncated to maxLength without appending an ellipsis.
//
// Parameters:
//
//	s         - The input string to be shortened.
//	maxLength - The maximum allowed length of the output string.
//
// Returns:
//
//	A string that is at most maxLength characters long, with an ellipsis appended if truncation occurred.
package stringutil

import "strings"

// Ellipsis shortens a string to a maximum length, adding "..." if truncated.
// Ellipsis truncates the input string s to a maximum length of maxLength characters.
// If the string exceeds maxLength, it is trimmed and an ellipsis ("...") is appended.
// Leading and trailing spaces are removed, and all newline characters are replaced with spaces.
// If maxLength is less than or equal to 3, the function returns the string truncated to maxLength without appending an ellipsis.
func Ellipsis(s string, maxLength int) string {
	s = strings.TrimSpace(s)             // Trim spaces first
	s = strings.ReplaceAll(s, "\n", " ") // Replace newlines with spaces for single line snippet
	s = strings.ReplaceAll(s, "\r", "")

	if maxLength < 0 {
		return ""
	}
	if len(s) <= maxLength {
		return s
	}
	if maxLength <= 3 { // Not enough space for "..."
		return s[:maxLength]
	}
	return s[:maxLength-3] + "..."
}
