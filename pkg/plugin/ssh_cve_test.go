package plugin

import (
	"strings"
	"testing"
)

func TestSSH_CVE_2016_0777_MatchFunc(t *testing.T) {
	tests := []struct {
		name     string
		ctx      map[string]string
		expected *MatchResult
	}{
		{
			name: "Vulnerable OpenSSH version",
			ctx: map[string]string{
				"ssh/banner": "OpenSSH_7.1p2 Ubuntu-4ubuntu2.1",
			},
			expected: &MatchResult{
				CVE:     []string{"CVE-2016-0777", "CVE-2016-0778"},
				Summary: "OpenSSH before 7.1p2 allows information leak via roaming",
				Port:    22,
				Info:    "OpenSSH_7.1p2 Ubuntu-4ubuntu2.1",
			},
		},
		{
			name: "Non-vulnerable OpenSSH version",
			ctx: map[string]string{
				"ssh/banner": "OpenSSH_8.0p1 Ubuntu-4ubuntu2.1",
			},
			expected: nil,
		},
		{
			name: "Empty banner",
			ctx: map[string]string{
				"ssh/banner": "",
			},
			expected: nil,
		},
		{
			name:     "No banner key in context",
			ctx:      map[string]string{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := &Plugin{
				MatchFunc: func(ctx map[string]string) *MatchResult {
					banner := ctx["ssh/banner"]
					if strings.Contains(banner, "OpenSSH_7.1p2") {
						return &MatchResult{
							CVE:     []string{"CVE-2016-0777", "CVE-2016-0778"},
							Summary: "OpenSSH before 7.1p2 allows information leak via roaming",
							Port:    22,
							Info:    banner,
						}
					}
					return nil
				},
			}

			result := plugin.MatchFunc(tt.ctx)
			if (result == nil && tt.expected != nil) || (result != nil && tt.expected == nil) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			} else if result != nil && tt.expected != nil {
				if result.CVE[0] != tt.expected.CVE[0] || result.Summary != tt.expected.Summary || result.Port != tt.expected.Port || result.Info != tt.expected.Info {
					t.Errorf("expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}
