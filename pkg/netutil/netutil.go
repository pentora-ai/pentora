// Package netutil provides utilities for parsing and expanding network targets and port specifications.
//
// It includes functions to:
//   - Expand target strings (IP addresses, CIDR blocks, IP ranges, or hostnames) into a flat list of unique, individual IP addresses.
//   - Parse and expand port strings (including ranges) into sorted, unique integer slices.
//   - Filter out non-targetable IPs such as multicast, unspecified, and link-local addresses.
//   - Safely increment IP addresses (IPv4 and IPv6) for range and CIDR expansion.
//   - Resolve hostnames to their corresponding IP addresses.
//
// Functions:
//
//   - ParseAndExpandTargets(targets []string) []string
//     Expands a list of target strings (IPs, hostnames, CIDRs, or ranges) into a unique list of IP addresses, filtering out non-scanable addresses.
//
//   - ParsePortString(portStr string) ([]int, error)
//     Parses a comma-separated string of ports and port ranges into a sorted, unique slice of integers.
//
//   - incIP(ip net.IP)
//     Increments an IP address in place (supports both IPv4 and IPv6).
//
//   - lookupAndAdd(target string, expandedIPs *[]string, seenIPs map[string]struct{})
//     Attempts to parse a target as an IP or resolve it as a hostname, adding unique results to the provided slice.
//
//   - filterNonScanableIPs(ips []string, alreadySeen map[string]struct{}) []string
//     Removes IPs that are generally not useful scan targets (e.g., multicast, unspecified, link-local).
package netutil

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	// For safe type casting if used elsewhere, though not directly in ParsePortString
)

// incIP increments an IP address (works for IPv4 and IPv6).
// It modifies the input IP slice in place.
func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 { // Break if the current byte didn't overflow
			break
		}
	}
}

// ParseAndExpandTargets expands a list of target strings (which can be IPs, hostnames,
// CIDR notations, or IP ranges) into a flat list of unique, individual IP address strings.
// It attempts to filter out common non-targetable IPs like multicast or unspecified addresses.
// Hostname resolution is performed if the target does not parse as an IP or CIDR.
func ParseAndExpandTargets(targets []string) []string {
	var expandedIPs []string
	seenIPs := make(map[string]struct{}) // To store unique IPs

	for _, t := range targets {
		target := strings.TrimSpace(t)
		if target == "" {
			continue
		}

		// Attempt to parse as CIDR first
		if strings.Contains(target, "/") {
			ipAddr, ipNet, err := net.ParseCIDR(target)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[WARN] ParseAndExpandTargets: Error parsing CIDR '%s': %v. Skipping.\n", target, err)
				continue
			}

			// Iterate over IP addresses in the CIDR network.
			currentIP := ipAddr.Mask(ipNet.Mask)
			for ipNet.Contains(currentIP) {
				ipToAdd := make(net.IP, len(currentIP))
				copy(ipToAdd, currentIP)
				ipStr := ipToAdd.String()

				if _, found := seenIPs[ipStr]; !found {
					// Filter network and broadcast for common IPv4 subnets (excluding /31, /32)
					isNetworkOrBroadcast := false
					if len(ipNet.Mask) == net.IPv4len && ipToAdd.To4() != nil {
						ones, bits := ipNet.Mask.Size()
						if bits == 32 && ones > 0 && ones < 31 {
							networkIP := ipNet.IP.To4()
							broadcastIP := make(net.IP, net.IPv4len)
							for i := 0; i < net.IPv4len; i++ {
								broadcastIP[i] = (ipNet.IP.To4())[i] | ^(ipNet.Mask)[i]
							}
							if ipToAdd.Equal(networkIP) || ipToAdd.Equal(broadcastIP) {
								isNetworkOrBroadcast = true
							}
						}
					}
					if !isNetworkOrBroadcast {
						expandedIPs = append(expandedIPs, ipStr)
						seenIPs[ipStr] = struct{}{}
					}
				}

				// Safety break for very large CIDRs
				if len(expandedIPs) > 1000000 && strings.Contains(target, "/") { // Increased limit
					fmt.Fprintf(os.Stderr, "[WARN] ParseAndExpandTargets: CIDR %s is extremely large, stopping expansion at %d IPs\n", target, len(expandedIPs))
					break
				}
				// Check if currentIP is the last IP in the network (all host bits are 1)
				// to prevent infinite loop on incIP for /0 or specific cases.
				isLastIP := true
				if ipToAdd.To4() != nil { // IPv4
					for i := 0; i < net.IPv4len; i++ {
						if (ipToAdd[i] | (^ipNet.Mask[i])) != 0xff {
							isLastIP = false
							break
						}
					}
				}
				//else if ipToAdd.To16() != nil { // IPv6
				// Similar logic for IPv6 if needed, more complex due to size.
				// For now, assume incIP handles IPv6 correctly.
				//}

				if isLastIP && ipNet.Contains(ipToAdd) { // If it's the broadcast/last address and still in network
					break // Stop for this CIDR
				}
				incIP(currentIP)
			}
		} else if strings.Contains(target, "-") { // IP Range
			parts := strings.SplitN(target, "-", 2)
			if len(parts) == 2 {
				startIPStr := strings.TrimSpace(parts[0])
				endIPStr := strings.TrimSpace(parts[1])
				startIP := net.ParseIP(startIPStr)
				var endIP net.IP

				// Handle simple last-octet range, e.g., "192.168.1.10-20"
				if startIP != nil && startIP.To4() != nil {
					endOctet, err := strconv.Atoi(endIPStr)
					if err == nil && endOctet >= 0 && endOctet <= 255 {
						ipBytes := startIP.To4()
						if ipBytes[3] <= byte(endOctet) { // Ensure range is valid
							baseIPStr := fmt.Sprintf("%d.%d.%d", ipBytes[0], ipBytes[1], ipBytes[2])
							for i := int(ipBytes[3]); i <= endOctet; i++ {
								ipStr := fmt.Sprintf("%s.%d", baseIPStr, i)
								if _, found := seenIPs[ipStr]; !found {
									expandedIPs = append(expandedIPs, ipStr)
									seenIPs[ipStr] = struct{}{}
								}
							}
							continue // Processed this simple range
						}
					}
				}
				// Handle full IP range, e.g., "192.168.1.10-192.168.1.20"
				endIP = net.ParseIP(endIPStr)
				if startIP != nil && endIP != nil {
					startIsV4 := startIP.To4() != nil
					endIsV4 := endIP.To4() != nil
					if startIsV4 != endIsV4 {
						fmt.Fprintf(os.Stderr, "[WARN] ParseAndExpandTargets: Mismatched IP versions in range '%s'. Skipping.\n", target)
						continue
					}

					compareResult := bytes.Compare(startIP, endIP)
					if (startIsV4 && bytes.Compare(startIP.To4(), endIP.To4()) > 0) || (!startIsV4 && compareResult > 0) {
						fmt.Fprintf(os.Stderr, "[WARN] ParseAndExpandTargets: Start IP is greater than End IP in range '%s'. Skipping.\n", target)
						continue
					}

					currentIP := make(net.IP, len(startIP))
					copy(currentIP, startIP)
					for {
						ipStr := currentIP.String()
						if _, found := seenIPs[ipStr]; !found {
							expandedIPs = append(expandedIPs, ipStr)
							seenIPs[ipStr] = struct{}{}
						}

						currentCompareVal := bytes.Compare(currentIP, endIP)
						if (startIsV4 && currentIP.Equal(endIP)) || (!startIsV4 && currentCompareVal == 0) {
							break // Reached endIP
						}
						incIP(currentIP)
						if (startIsV4 && bytes.Compare(currentIP.To4(), startIP.To4()) < 0) || (!startIsV4 && bytes.Compare(currentIP, startIP) < 0) {
							fmt.Fprintf(os.Stderr, "[WARN] ParseAndExpandTargets: IP range %s wrapped around. Stopping.\n", target)
							break // Wrapped around
						}
						if len(expandedIPs) > 262144 && strings.Contains(target, "-") { // Increased safety limit for ranges
							fmt.Fprintf(os.Stderr, "[WARN] ParseAndExpandTargets: IP range %s is very large, stopping expansion at %d IPs\n", target, len(expandedIPs))
							break
						}
					}
				} else {
					fmt.Fprintf(os.Stderr, "[WARN] ParseAndExpandTargets: Invalid IP address in range: '%s'. Attempting hostname lookup.\n", target)
					// Fall through to hostname lookup if range parsing fails but it's not a CIDR
					lookupAndAdd(target, &expandedIPs, seenIPs)
				}
			} else { // Not CIDR, not a recognized range format containing '-' -> Treat as single IP or hostname
				lookupAndAdd(target, &expandedIPs, seenIPs)
			}
		} else { // Single IP or hostname
			lookupAndAdd(target, &expandedIPs, seenIPs)
		}
	}
	return filterNonScanableIPs(expandedIPs, seenIPs) // Use a new map for filtering stage
}

// lookupAndAdd attempts to parse as IP, then as hostname.
func lookupAndAdd(target string, expandedIPs *[]string, seenIPs map[string]struct{}) {
	ip := net.ParseIP(target)
	if ip != nil {
		ipStr := ip.String()
		if _, found := seenIPs[ipStr]; !found {
			*expandedIPs = append(*expandedIPs, ipStr)
			seenIPs[ipStr] = struct{}{}
		}
		return
	}

	// If not a direct IP, try DNS lookup
	addrs, err := net.LookupHost(target)
	if err == nil {
		for _, addr := range addrs {
			// Ensure resolved address is a valid IP and not already seen
			resolvedIP := net.ParseIP(addr)
			if resolvedIP != nil {
				ipStr := resolvedIP.String()
				if _, found := seenIPs[ipStr]; !found {
					*expandedIPs = append(*expandedIPs, ipStr)
					seenIPs[ipStr] = struct{}{}
				}
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "[WARN] ParseAndExpandTargets: Could not parse or resolve target '%s': %v. Skipping.\n", target, err)
	}
}

// filterNonScanableIPs removes IPs that are generally not useful targets.
// Loopback is handled by module config, this filters others.
func filterNonScanableIPs(ips []string, _ map[string]struct{}) []string {
	// This function assumes `alreadySeen` was used to populate `ips` with unique strings.
	// If `ips` might contain duplicates or IPs that were added bypassing `seenIPs` in `ParseAndExpandTargets`,
	// it's better to re-initialize `seen` here or pass `seenIPs` by reference and modify it.
	// For clarity and safety, let's re-filter based on the content of `ips`.

	var result []string
	finalSeen := make(map[string]struct{}) // Use a new map for this filtering stage

	for _, ipStr := range ips {
		trimmedIPStr := strings.TrimSpace(ipStr)
		if trimmedIPStr == "" {
			continue
		}

		ip := net.ParseIP(trimmedIPStr)
		if ip == nil ||
			ip.IsMulticast() ||
			ip.IsUnspecified() ||
			ip.IsLinkLocalUnicast() || // Typically 169.254.x.x, fe80::/10
			ip.IsLinkLocalMulticast() { // ff02::/16
			// Loopback (127.0.0.0/8, ::1) filtering should be based on module's `AllowLoopback` config.
			// If it's critical to filter here always (e.g. results of DNS lookup), add:
			// || ip.IsLoopback()
			continue
		}
		if _, found := finalSeen[trimmedIPStr]; !found {
			result = append(result, trimmedIPStr)
			finalSeen[trimmedIPStr] = struct{}{}
		}
	}
	return result
}

// ParsePortString parses a comma-separated string of ports and port ranges
// into a slice of unique integers, sorted.
// Example: "80,443,1000-1002,22" -> [22, 80, 443, 1000, 1001, 1002]
func ParsePortString(portStr string) ([]int, error) {
	if strings.TrimSpace(portStr) == "" {
		return []int{}, nil // Return empty slice for empty or whitespace-only string
	}

	seenPorts := make(map[int]struct{})
	var ports []int

	parts := strings.Split(portStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "-") { // Port range
			rangeParts := strings.SplitN(part, "-", 2)
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid port range format: '%s'", part)
			}
			startStr, endStr := strings.TrimSpace(rangeParts[0]), strings.TrimSpace(rangeParts[1])

			start, err := strconv.Atoi(startStr)
			if err != nil {
				return nil, fmt.Errorf("invalid start port in range '%s': %w", part, err)
			}
			end, err := strconv.Atoi(endStr)
			if err != nil {
				return nil, fmt.Errorf("invalid end port in range '%s': %w", part, err)
			}

			if start < 0 || start > 65535 || end < 0 || end > 65535 {
				return nil, fmt.Errorf("port numbers in range '%s' must be between 0 and 65535", part)
			}
			if start > end {
				return nil, fmt.Errorf("start port %d cannot be greater than end port %d in range '%s'", start, end, part)
			}

			for i := start; i <= end; i++ {
				if _, found := seenPorts[i]; !found {
					ports = append(ports, i)
					seenPorts[i] = struct{}{}
				}
			}
		} else { // Single port
			port, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid port number '%s': %w", part, err)
			}
			if port < 0 || port > 65535 {
				return nil, fmt.Errorf("port number '%d' must be between 0 and 65535", port)
			}
			if _, found := seenPorts[port]; !found {
				ports = append(ports, port)
				seenPorts[port] = struct{}{}
			}
		}
	}
	sort.Ints(ports) // Sort for consistent output and easier processing later
	return ports, nil
}
