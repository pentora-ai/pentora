package plugin

import (
	"strings"
)

func init() {
	Register(&Plugins{
		ID:           "ssh_cve_2016_0777",
		Name:         "OpenSSH 7.1p2 Vulnerability",
		RequirePorts: []int{22},
		RequireKeys:  []string{"ssh/banner"},
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
	})
}
