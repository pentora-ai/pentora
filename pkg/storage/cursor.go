package storage

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// Cursor represents a pagination cursor for scan listing.
// It contains the last scan ID and timestamp to enable efficient pagination.
type Cursor struct {
	LastScanID string `json:"id"`
	LastTime   int64  `json:"ts"` // Unix timestamp in nanoseconds
}

// EncodeCursor encodes a cursor to a base64 URL-safe string.
// Returns empty string if cursor is nil or invalid.
func EncodeCursor(c *Cursor) string {
	if c == nil || c.LastScanID == "" {
		return ""
	}

	data, err := json.Marshal(c)
	if err != nil {
		return ""
	}

	return base64.URLEncoding.EncodeToString(data)
}

// DecodeCursor decodes a base64-encoded cursor string.
// Returns nil and no error for empty cursor (first page).
// Returns error if cursor is malformed.
func DecodeCursor(encoded string) (*Cursor, error) {
	if encoded == "" {
		return nil, nil // First page
	}

	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor encoding: %w", err)
	}

	var c Cursor
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("invalid cursor format: %w", err)
	}

	if c.LastScanID == "" {
		return nil, fmt.Errorf("invalid cursor: missing scan ID")
	}

	return &c, nil
}
