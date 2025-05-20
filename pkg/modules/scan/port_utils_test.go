package scan

import (
	"reflect"
	"testing"
)

func TestParsePortsString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      []int
		expectErr bool
	}{
		{
			name:      "empty string",
			input:     "",
			want:      []int{},
			expectErr: false,
		},
		{
			name:      "single port",
			input:     "22",
			want:      []int{22},
			expectErr: false,
		},
		{
			name:      "multiple ports",
			input:     "22,80,443",
			want:      []int{22, 80, 443},
			expectErr: false,
		},
		{
			name:      "port range",
			input:     "1000-1003",
			want:      []int{1000, 1001, 1002, 1003},
			expectErr: false,
		},
		{
			name:      "mixed ports and ranges",
			input:     "22,80,1000-1002,443",
			want:      []int{22, 80, 1000, 1001, 1002, 443},
			expectErr: false,
		},
		{
			name:      "duplicate ports",
			input:     "22,22,80,80,1000-1002,1001",
			want:      []int{22, 80, 1000, 1001, 1002},
			expectErr: false,
		},
		{
			name:      "invalid port number",
			input:     "22,abc,80",
			want:      nil,
			expectErr: true,
		},
		{
			name:      "invalid range format",
			input:     "22,80-",
			want:      nil,
			expectErr: true,
		},
		{
			name:      "range with invalid numbers",
			input:     "22,80-abc",
			want:      nil,
			expectErr: true,
		},
		{
			name:      "range with start > end",
			input:     "100-90",
			want:      nil,
			expectErr: true,
		},
		{
			name:      "port below 1",
			input:     "0,22",
			want:      nil,
			expectErr: true,
		},
		{
			name:      "port above 65535",
			input:     "22,70000",
			want:      nil,
			expectErr: true,
		},
		{
			name:      "range above 65535",
			input:     "65534-65536",
			want:      nil,
			expectErr: true,
		},
		{
			name:      "spaces and empty parts",
			input:     " 22 , , 80 ,1000-1001 ",
			want:      []int{22, 80, 1000, 1001},
			expectErr: false,
		},
		{
			name:      "all invalid",
			input:     "abc,,--,0-0",
			want:      nil,
			expectErr: true,
		},
		{
			name:      "range with single port",
			input:     "100-100",
			want:      []int{100},
			expectErr: false,
		},
		{
			name:      "multiple ranges",
			input:     "1000-1001-1002",
			want:      nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePortsString(tt.input)
			if (err != nil) != tt.expectErr {
				t.Errorf("parsePortsString() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parsePortsString() = %v, want %v", got, tt.want)
			}
		})
	}
}
