package netutil

import (
	"net"
	"reflect"
	"testing"
)

func TestParsePortString(t *testing.T) {
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
			input:     "80",
			want:      []int{80},
			expectErr: false,
		},
		{
			name:      "multiple ports",
			input:     "80,443,22",
			want:      []int{22, 80, 443},
			expectErr: false,
		},
		{
			name:      "port range",
			input:     "1000-1002",
			want:      []int{1000, 1001, 1002},
			expectErr: false,
		},
		{
			name:      "mixed ports and ranges",
			input:     "80,443,1000-1002,22",
			want:      []int{22, 80, 443, 1000, 1001, 1002},
			expectErr: false,
		},
		{
			name:      "duplicate ports",
			input:     "80,80,443,443",
			want:      []int{80, 443},
			expectErr: false,
		},
		{
			name:      "duplicate in range and single",
			input:     "80,78-82",
			want:      []int{78, 79, 80, 81, 82},
			expectErr: false,
		},
		{
			name:      "whitespace handling",
			input:     " 80 ,  443 , 1000 - 1002 , 22 ",
			want:      []int{22, 80, 443, 1000, 1001, 1002},
			expectErr: false,
		},
		{
			name:      "invalid port number",
			input:     "abc",
			want:      nil,
			expectErr: true,
		},
		{
			name:      "invalid port range format",
			input:     "80-",
			want:      nil,
			expectErr: true,
		},
		{
			name:      "start greater than end in range",
			input:     "100-90",
			want:      nil,
			expectErr: true,
		},
		{
			name:      "port below 0",
			input:     "-1",
			want:      nil,
			expectErr: true,
		},
		{
			name:      "port above 65535",
			input:     "65536",
			want:      nil,
			expectErr: true,
		},
		{
			name:      "range with port above 65535",
			input:     "65534-65536",
			want:      nil,
			expectErr: true,
		},
		{
			name:      "range with port below 0",
			input:     "-2-2",
			want:      nil,
			expectErr: true,
		},
		{
			name:      "empty elements",
			input:     "80,,443",
			want:      []int{80, 443},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePortString(tt.input)
			if (err != nil) != tt.expectErr {
				t.Errorf("ParsePortString() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParsePortString() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestFilterNonScanableIPs(t *testing.T) {
	tests := []struct {
		name        string
		input       []string
		want        []string
		alreadySeen map[string]struct{}
	}{
		{
			name:        "valid IPv4 addresses",
			input:       []string{"192.168.1.1", "8.8.8.8"},
			want:        []string{"192.168.1.1", "8.8.8.8"},
			alreadySeen: map[string]struct{}{},
		},
		{
			name:        "filters multicast IPv4",
			input:       []string{"224.0.0.1", "239.255.255.250", "8.8.8.8"},
			want:        []string{"8.8.8.8"},
			alreadySeen: map[string]struct{}{},
		},
		{
			name:        "filters unspecified IPv4",
			input:       []string{"0.0.0.0", "8.8.8.8"},
			want:        []string{"8.8.8.8"},
			alreadySeen: map[string]struct{}{},
		},
		{
			name:        "filters link-local unicast IPv4",
			input:       []string{"169.254.1.1", "8.8.8.8"},
			want:        []string{"8.8.8.8"},
			alreadySeen: map[string]struct{}{},
		},
		{
			name:        "filters link-local multicast IPv6",
			input:       []string{"ff02::1", "2001:db8::1"},
			want:        []string{"2001:db8::1"},
			alreadySeen: map[string]struct{}{},
		},
		{
			name:        "filters unspecified IPv6",
			input:       []string{"::", "2001:db8::1"},
			want:        []string{"2001:db8::1"},
			alreadySeen: map[string]struct{}{},
		},
		{
			name:        "filters link-local unicast IPv6",
			input:       []string{"fe80::1", "2001:db8::1"},
			want:        []string{"2001:db8::1"},
			alreadySeen: map[string]struct{}{},
		},
		{
			name:        "filters invalid IPs and empty strings",
			input:       []string{"not-an-ip", "", "8.8.8.8"},
			want:        []string{"8.8.8.8"},
			alreadySeen: map[string]struct{}{},
		},
		{
			name:        "removes duplicates",
			input:       []string{"8.8.8.8", "8.8.8.8", "1.1.1.1"},
			want:        []string{"8.8.8.8", "1.1.1.1"},
			alreadySeen: map[string]struct{}{},
		},
		{
			name:        "trims whitespace",
			input:       []string{" 8.8.8.8 ", "\t1.1.1.1\n"},
			want:        []string{"8.8.8.8", "1.1.1.1"},
			alreadySeen: map[string]struct{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterNonScanableIPs(tt.input, tt.alreadySeen)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filterNonScanableIPs() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestIncIP(t *testing.T) {
	tests := []struct {
		name     string
		input    net.IP
		expected net.IP
	}{
		{
			name:     "increment IPv4 address",
			input:    net.IP{192, 168, 1, 1},
			expected: net.IP{192, 168, 1, 2},
		},
		{
			name:     "increment IPv4 with carry",
			input:    net.IP{192, 168, 1, 255},
			expected: net.IP{192, 168, 2, 0},
		},
		{
			name:     "increment IPv4 max",
			input:    net.IP{255, 255, 255, 255},
			expected: net.IP{0, 0, 0, 0},
		},
		{
			name:     "increment IPv6 address",
			input:    net.ParseIP("2001:db8::1"),
			expected: net.ParseIP("2001:db8::2"),
		},
		{
			name:     "increment IPv6 with carry",
			input:    net.ParseIP("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"),
			expected: net.ParseIP("::"),
		},
		{
			name:     "increment IPv6 with mid-carry",
			input:    net.ParseIP("2001:db8::ffff"),
			expected: net.ParseIP("2001:db8::1:0"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ipCopy := make(net.IP, len(tt.input))
			copy(ipCopy, tt.input)
			incIP(ipCopy)
			if !ipCopy.Equal(tt.expected) {
				t.Errorf("incIP(%v) = %v, want %v", tt.input, ipCopy, tt.expected)
			}
		})
	}
}
func TestParseAndExpandTargets(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
		// Note: The output order is not guaranteed, so we'll use a map for comparison.
	}{
		{
			name:  "single IPv4 address",
			input: []string{"8.8.8.8"},
			want:  []string{"8.8.8.8"},
		},
		{
			name:  "multiple IPv4 addresses with whitespace",
			input: []string{" 8.8.8.8 ", "1.1.1.1"},
			want:  []string{"8.8.8.8", "1.1.1.1"},
		},
		{
			name:  "duplicate IPv4 addresses",
			input: []string{"8.8.8.8", "8.8.8.8"},
			want:  []string{"8.8.8.8"},
		},
		{
			name:  "IPv4 CIDR /30 (filters network and broadcast)",
			input: []string{"192.168.1.0/30"},
			want:  []string{"192.168.1.1", "192.168.1.2"},
		},
		{
			name:  "IPv4 CIDR /31 (no filtering, both addresses included)",
			input: []string{"192.168.1.0/31"},
			want:  []string{"192.168.1.0", "192.168.1.1"},
		},
		{
			name:  "IPv4 CIDR /32 (single address)",
			input: []string{"192.168.1.5/32"},
			want:  []string{"192.168.1.5"},
		},
		/*{
			name:  "IPv6 CIDR /126 (should include all 4 addresses)",
			input: []string{"2001:db8::/126"},
			want:  []string{"2001:db8::", "2001:db8::1", "2001:db8::2", "2001:db8::3"},
		},*/
		{
			name:  "simple last-octet IPv4 range",
			input: []string{"192.168.1.10-12"},
			want:  []string{"192.168.1.10", "192.168.1.11", "192.168.1.12"},
		},
		{
			name:  "full IPv4 range",
			input: []string{"192.168.1.10-192.168.1.12"},
			want:  []string{"192.168.1.10", "192.168.1.11", "192.168.1.12"},
		},
		/*{
			name:  "IPv6 range",
			input: []string{"2001:db8::1-2001:db8::3"},
			want:  []string{"2001:db8::1", "2001:db8::2", "2001:db8::3"},
		},*/
		{
			name:  "mix of IPs, CIDR, and ranges",
			input: []string{"8.8.8.8", "192.168.1.10-12", "192.168.1.0/30"},
			want:  []string{"8.8.8.8", "192.168.1.10", "192.168.1.11", "192.168.1.12", "192.168.1.1", "192.168.1.2"},
		},
		{
			name:  "filters multicast, unspecified, and link-local",
			input: []string{"224.0.0.1", "0.0.0.0", "169.254.1.1", "8.8.8.8"},
			want:  []string{"8.8.8.8"},
		},
		{
			name:  "invalid IP and hostname (should skip invalid, try DNS for hostname)",
			input: []string{"not-an-ip", "8.8.8.8"},
			want:  []string{"8.8.8.8"},
		},
		{
			name:  "empty and whitespace-only elements",
			input: []string{"", "   ", "8.8.8.8"},
			want:  []string{"8.8.8.8"},
		},
		{
			name:  "range with start > end (should skip)",
			input: []string{"192.168.1.12-192.168.1.10", "8.8.8.8"},
			want:  []string{"8.8.8.8"},
		},
		{
			name:  "range with mismatched IP versions (should skip)",
			input: []string{"192.168.1.10-2001:db8::1", "8.8.8.8"},
			want:  []string{"8.8.8.8"},
		},
	}

	// Helper to compare slices as sets
	compareStringSets := func(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		ma := make(map[string]struct{}, len(a))
		for _, v := range a {
			ma[v] = struct{}{}
		}
		for _, v := range b {
			if _, ok := ma[v]; !ok {
				return false
			}
		}
		return true
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseAndExpandTargets(tt.input)
			if !compareStringSets(got, tt.want) {
				t.Errorf("ParseAndExpandTargets(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
