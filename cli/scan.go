// cli/scan.go
package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pentoraai/pentora/scanner"
	"github.com/spf13/cobra"
)

// portList is a CLI flag that allows the user to define custom ports to scan
var portList string
var discover bool

// ScanCmd defines the 'scan' command entry point for the CLI
var ScanCmd = &cobra.Command{
	Use:   "scan [host]",
	Short: "Start a scan on given target",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		host := args[0]
		ports := []int{22, 80, 443} // Default ports
		if portList != "" {
			ports = parsePorts(portList)
		} else if discover {
			fmt.Printf("Discovering open ports on %s (range: 1-1000)...\n", host)
			ports = scanner.DiscoverPorts(host, 1, 1000)
			if len(ports) == 0 {
				fmt.Println("No open ports found.")
				return
			}
		}

		vulnScan, _ := cmd.Flags().GetBool("vuln")

		// Construct the scan job
		job := scanner.ScanJob{
			Targets:        []string{host},
			Ports:          ports,
			EnableVulnScan: vulnScan,
		}

		fmt.Printf("Scanning %s on ports %v...\n", host, ports)

		results, err := scanner.Run(job)
		if err != nil {
			fmt.Printf("Error during scan: %v\n", err)
			return
		}

		// Print each result from the scan
		for _, r := range results {
			if r.Status == "open" {
				fmt.Printf(" - %s:%d → %s (%s %s)", r.IP, r.Port, r.Status, r.Service, r.Version)
				if len(r.CVEs) > 0 {
					fmt.Printf(" [CVE: %s]", strings.Join(r.CVEs, ", "))
				}
				fmt.Println()
			} else {
				fmt.Printf(" - %s:%d → %s\n", r.IP, r.Port, r.Status)
			}
		}
	},
}

// Register CLI flags
func init() {
	ScanCmd.Flags().StringVarP(&portList, "ports", "p", "", "Comma-separated list of ports to scan (e.g. 22,80,443)")
	ScanCmd.Flags().BoolVarP(&discover, "discover", "d", false, "Automatically discover open ports in the range 1-1000")
	ScanCmd.Flags().Bool("vuln", false, "Enable vulnerability matching against banners")
}

// parsePorts parses a comma-separated string of ports into a slice of ints
func parsePorts(input string) []int {
	parts := strings.Split(input, ",")
	var ports []int
	for _, p := range parts {
		if port, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
			ports = append(ports, port)
		}
	}
	return ports
}
