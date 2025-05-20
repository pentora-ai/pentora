// pkg/modules/scan/port_utils.go
package scan

import (
	"fmt"
	"strconv"
	"strings"
)

// parsePortsString converts a comma-separated string of ports and ranges
// (e.g., "22,80,1000-1024,3306") into a slice of integers.
func parsePortsString(portsStr string) ([]int, error) {
	if portsStr == "" {
		return []int{}, nil
	}
	seenPorts := make(map[int]bool)
	var result []int

	parts := strings.Split(portsStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "-") { // Range
			rangeParts := strings.SplitN(part, "-", 2)
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid port range format: '%s'", part)
			}
			startPort, err1 := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			endPort, err2 := strconv.Atoi(strings.TrimSpace(rangeParts[1]))

			if err1 != nil || err2 != nil || startPort < 1 || endPort > 65535 || startPort > endPort {
				return nil, fmt.Errorf("invalid port numbers in range: '%s'", part)
			}
			for p := startPort; p <= endPort; p++ {
				if !seenPorts[p] {
					result = append(result, p)
					seenPorts[p] = true
				}
			}
		} else { // Single port
			port, err := strconv.Atoi(part)
			if err != nil || port < 1 || port > 65535 {
				return nil, fmt.Errorf("invalid port number: '%s'", part)
			}
			if !seenPorts[port] {
				result = append(result, port)
				seenPorts[port] = true
			}
		}
	}
	if len(result) == 0 && portsStr != "" {
		return nil, fmt.Errorf("no valid ports parsed from string: '%s'", portsStr)
	}
	return result, nil
}
