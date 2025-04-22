package main

import (
	"fmt"

	"github.com/pentoraai/pentora/plugin"
)

func main() {
	ctx := map[string]string{
		"ssh/banner": "SSH-2.0-OpenSSH_7.1p2 Ubuntu-3ubuntu0.1",
	}
	openPorts := []int{22}
	satisfied := []string{}

	results := plugin.MatchAll(ctx, openPorts, satisfied)

	for _, res := range results {
		fmt.Printf("\n🛡️  CVE match on port %d:\n", res.Port)
		fmt.Printf(" - CVEs: %v\n", res.CVE)
		fmt.Printf(" - Summary: %s\n", res.Summary)
		fmt.Printf(" - Info: %s\n", res.Info)
	}
}
