// pkg/utils/networkutils.go

package utils

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/spf13/cast"
)

// parseAndExpandTargets expands targets (CIDRs, ranges) into a list of individual IP strings.
// It should be robust enough for common network notations.
func ParseAndExpandTargets(targets []string) []string {
	var expanded []string
	for _, t := range targets {
		target := strings.TrimSpace(t)
		if target == "" {
			continue
		}

		if strings.Contains(target, "/") { // CIDR notation
			ipAddr, ipNet, err := net.ParseCIDR(target)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[WARN] Module 'icmp-ping-discovery': Error parsing CIDR '%s': %v. Skipping.\n", target, err)
				continue
			}
			// Iterate over IP addresses in the CIDR network.
			for ip := ipAddr.Mask(ipNet.Mask); ipNet.Contains(ip); incIP(ip) {
				// Create a copy to avoid modifying the loop variable if it's a slice.
				ipToAdd := make(net.IP, len(ip))
				copy(ipToAdd, ip)

				// Attempt to filter out network and broadcast addresses for common IPv4 subnets.
				// This is a simplification. For /31, /32, both IPs are usable. IPv6 has no broadcast.
				isNetworkOrBroadcast := false
				if len(ipNet.Mask) == net.IPv4len && ip.To4() != nil {
					ones, bits := ipNet.Mask.Size()
					if bits == 32 && ones > 0 && ones < 31 { // Typically for /1 to /30 masks
						networkIP := ipNet.IP.To4()
						broadcastIP := make(net.IP, net.IPv4len)
						for i := 0; i < net.IPv4len; i++ {
							broadcastIP[i] = (ipNet.IP.To4())[i] | ^(ipNet.Mask)[i]
						}
						if ip.Equal(networkIP) || ip.Equal(broadcastIP) {
							isNetworkOrBroadcast = true
						}
					}
				}

				if !isNetworkOrBroadcast {
					expanded = append(expanded, ipToAdd.String())
				}

				// Safety break for very large CIDRs to prevent excessive memory usage/time
				if len(expanded)%65536 == 0 && len(expanded) > 0 && strings.Contains(target, "/") {
					fmt.Fprintf(os.Stderr, "[WARN] Module 'icmp-ping-discovery': CIDR %s is very large or incIP is stuck, stopping expansion at %d IPs\n", target, len(expanded))
					if len(expanded) > 200000 { // Harder limit
						break
					}
				}
				// Check if we have looped around in the CIDR (e.g. incIP on last IP)
				if len(ipNet.Mask) == net.IPv4len && ip.To4() != nil && ip.Equal(net.IPv4bcast) && ipNet.Contains(ip) {
					// If current IP is 255.255.255.255 and still in network, likely a /0 or error in incIP
					if bytes.Compare(ip.Mask(ipNet.Mask), ipNet.IP.Mask(ipNet.Mask)) == 0 { // Still in the same network start
						// This condition might be too broad, but serves as a safety for large networks.
						// Ideally, break if ip becomes less than start ip after masking for the next iteration
					}
				}

			}
		} else if strings.Contains(target, "-") { // Basic range support
			parts := strings.SplitN(target, "-", 2)
			if len(parts) == 2 {
				startIPStr := strings.TrimSpace(parts[0])
				endIPStr := strings.TrimSpace(parts[1])
				startIP := net.ParseIP(startIPStr)
				endIP := net.ParseIP(endIPStr)

				// Try to parse simple last-octet range like "192.168.1.10-20"
				if startIP == nil && endIP == nil {
					baseParts := strings.Split(startIPStr, ".")
					if len(baseParts) == 4 {
						startOctet, errStart := cast.ToIntE(baseParts[3])
						endOctet, errEnd := cast.ToIntE(endIPStr)
						if errStart == nil && errEnd == nil && endOctet >= startOctet && endOctet <= 255 && startOctet >= 0 && startOctet <= 255 {
							baseIPStr := strings.Join(baseParts[:3], ".")
							for i := startOctet; i <= endOctet; i++ {
								expanded = append(expanded, fmt.Sprintf("%s.%d", baseIPStr, i))
							}
							continue // Processed this simple range, move to next target
						}
					}
				}

				// Handle full IP range like "192.168.1.10-192.168.1.20"
				if startIP != nil && endIP != nil {
					// Ensure IPs are of the same family (IPv4 or IPv6) and startIP <= endIP
					startIsV4 := startIP.To4() != nil
					endIsV4 := endIP.To4() != nil
					if startIsV4 != endIsV4 {
						fmt.Fprintf(os.Stderr, "[WARN] Module 'icmp-ping-discovery': Mismatched IP versions in range '%s'. Skipping.\n", target)
						continue
					}

					compareResult := bytes.Compare(startIP, endIP)
					if startIsV4 { // Use To4 for comparison if they are v4
						compareResult = bytes.Compare(startIP.To4(), endIP.To4())
					}

					if compareResult <= 0 {
						currentIP := make(net.IP, len(startIP))
						copy(currentIP, startIP)
						for {
							ipToAdd := make(net.IP, len(currentIP))
							copy(ipToAdd, currentIP)
							expanded = append(expanded, ipToAdd.String())

							currentCompareResult := bytes.Compare(currentIP, endIP)
							if startIsV4 {
								currentCompareResult = bytes.Compare(currentIP.To4(), endIP.To4())
							}

							if currentCompareResult == 0 { // Reached endIP
								break
							}
							incIP(currentIP)
							// Safety break for very large ranges or potential infinite loops
							if len(expanded) > 20000 && strings.Contains(target, "-") { // Adjusted limit
								fmt.Fprintf(os.Stderr, "[WARN] Module 'icmp-ping-discovery': IP range %s is large, stopping expansion at %d IPs\n", target, len(expanded))
								break
							}
							// Additional check to prevent infinite loop if incIP wraps around incorrectly for the given range.
							if bytes.Compare(currentIP, startIP) < 0 && len(startIP) == len(currentIP) { // Wrapped around
								fmt.Fprintf(os.Stderr, "[WARN] Module 'icmp-ping-discovery': IP range %s seems to have wrapped around. Stopping.\n", target)
								break
							}
						}
					} else {
						fmt.Fprintf(os.Stderr, "[WARN] Module 'icmp-ping-discovery': Start IP is greater than End IP in range '%s'. Skipping.\n", target)
					}
				} else {
					fmt.Fprintf(os.Stderr, "[WARN] Module 'icmp-ping-discovery': Invalid IP address in range: '%s'. Skipping.\n", target)
				}
			} else { // Not CIDR, not a recognized range format containing '-'
				expanded = append(expanded, target)
			}
		} else { // Single IP
			expanded = append(expanded, target)
		}
	}
	return uniqueAndFilterSpecialIPs(expanded)
}

// incIP increments an IP address (works for IPv4 and IPv6).
func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// uniqueAndFilterSpecialIPs removes duplicates and filters out loopback, multicast, link-local, and unspecified IPs.
func uniqueAndFilterSpecialIPs(ips []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, ipStr := range ips {
		trimmedIPStr := strings.TrimSpace(ipStr)
		if trimmedIPStr == "" || seen[trimmedIPStr] {
			continue
		}

		ip := net.ParseIP(trimmedIPStr)
		// Note: Loopback filtering is now handled by the module's 'AllowLoopback' config in Execute.
		// Here we filter other generally non-targetable IPs.
		if ip == nil ||
			ip.IsMulticast() ||
			ip.IsUnspecified() ||
			ip.IsLinkLocalUnicast() ||
			ip.IsLinkLocalMulticast() {
			continue
		}
		seen[trimmedIPStr] = true
		result = append(result, trimmedIPStr)
	}
	return result
}
