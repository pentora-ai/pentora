package stringutil

import (
	"testing"
)

func TestEllipsis(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLength int
		expected  string
	}{
		{
			name:      "No truncation needed",
			input:     "hello world",
			maxLength: 20,
			expected:  "hello world",
		},
		{
			name:      "Truncate with ellipsis",
			input:     "The quick brown fox jumps over the lazy dog",
			maxLength: 16,
			expected:  "The quick bro...",
		},
		{
			name:      "Truncate with maxLength less than or equal to 3",
			input:     "abcdefg",
			maxLength: 3,
			expected:  "abc",
		},
		{
			name:      "String with leading and trailing spaces",
			input:     "   padded string   ",
			maxLength: 10,
			expected:  "padded ...",
		},
		{
			name:      "String with newlines and carriage returns",
			input:     "foo\nbar\r\nbaz",
			maxLength: 10,
			expected:  "foo bar...",
		},
		{
			name:      "Empty string",
			input:     "",
			maxLength: 5,
			expected:  "",
		},
		{
			name:      "maxLength zero",
			input:     "something",
			maxLength: 0,
			expected:  "",
		},
		{
			name:      "maxLength negative",
			input:     "something",
			maxLength: -1,
			expected:  "",
		},
		{
			name:      "String exactly maxLength",
			input:     "12345",
			maxLength: 5,
			expected:  "12345",
		},
		{
			name:      "String with only spaces and newlines",
			input:     "   \n\r   ",
			maxLength: 2,
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Ellipsis(tt.input, tt.maxLength)
			if result != tt.expected {
				t.Errorf("Ellipsis(%q, %d) = %q; want %q",
					tt.input, tt.maxLength, result, tt.expected)
			}
		})
	}
}