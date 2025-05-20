package scanner

import (
	"fmt"
	"log"
	"net"
	"time"
)

// DiscoverPorts scans a range of ports on a single target
// and returns only the open ports.
func DiscoverPorts(ip string, startPort, endPort int) []int {
	openPorts := []int{}
	timeout := 500 * time.Millisecond

	for port := startPort; port <= endPort; port++ {
		address := net.JoinHostPort(ip, itoa(port))
		conn, err := net.DialTimeout("tcp", address, timeout)
		if err == nil {
			openPorts = append(openPorts, port)
			if err := conn.Close(); err != nil {
				log.Printf("connection close failed (addr: %s): %v", conn.RemoteAddr(), err)
			}
		}
	}

	return openPorts
}

// itoa converts an int to string without importing strconv
func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
