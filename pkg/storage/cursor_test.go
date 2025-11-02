package storage

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// ✅ happy path – encode and decode roundtrip
func TestCursor_EncodeDecode_Success(t *testing.T) {
	c := &Cursor{LastScanID: "scan-123", LastTime: 1234567890}
	encoded := EncodeCursor(c)
	require.NotEmpty(t, encoded)

	decoded, err := DecodeCursor(encoded)
	require.NoError(t, err)
	require.Equal(t, c.LastScanID, decoded.LastScanID)
	require.Equal(t, c.LastTime, decoded.LastTime)
}

// ✅ nil cursor returns empty string
func TestCursor_Encode_NilCursor(t *testing.T) {
	require.Equal(t, "", EncodeCursor(nil))
}

// ✅ empty ID returns empty string
func TestCursor_Encode_EmptyID(t *testing.T) {
	c := &Cursor{LastScanID: "", LastTime: 123}
	require.Equal(t, "", EncodeCursor(c))
}

// TestCursor_Encode_JSONError removed - can't trigger marshal error with normal Cursor struct
// EncodeCursor silently returns empty string on marshal error (defensive programming)

// ✅ empty string input -> first page (returns nil, nil)
func TestCursor_Decode_EmptyString(t *testing.T) {
	c, err := DecodeCursor("")
	require.NoError(t, err)
	require.Nil(t, c)
}

// ✅ invalid base64 string
func TestCursor_Decode_InvalidBase64(t *testing.T) {
	c, err := DecodeCursor("%%%not-base64%%%")
	require.Error(t, err)
	require.Nil(t, c)
	require.Contains(t, err.Error(), "invalid cursor encoding")
}

// ✅ invalid JSON structure
func TestCursor_Decode_InvalidJSON(t *testing.T) {
	// base64 of plain string, not JSON
	encoded := base64.URLEncoding.EncodeToString([]byte("not-json"))
	c, err := DecodeCursor(encoded)
	require.Error(t, err)
	require.Nil(t, c)
	require.Contains(t, err.Error(), "invalid cursor format")
}

// ✅ missing LastScanID
func TestCursor_Decode_MissingScanID(t *testing.T) {
	c := Cursor{LastScanID: "", LastTime: 111}
	data, _ := json.Marshal(c)
	encoded := base64.URLEncoding.EncodeToString(data)

	out, err := DecodeCursor(encoded)
	require.Error(t, err)
	require.Nil(t, out)
	require.Contains(t, err.Error(), "missing scan ID")
}
