// scanner/scanner.go
package scanner

import (
	"fmt"
	"net"
	"time"
)

func ScanPort(host string, port int) bool {
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func GrabBanner(host string, port int) (string, error) {
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return "", err
	}
	return string(buffer[:n]), nil
}

func HTTPProbe(host string, port int) (string, error) {
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	req := "GET / HTTP/1.1\r\nHost: " + host + "\r\nConnection: close\r\n\r\n"
	_, err = conn.Write([]byte(req))
	if err != nil {
		return "", err
	}

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buffer := make([]byte, 2048)
	n, err := conn.Read(buffer)
	if err != nil {
		return "", err
	}
	return string(buffer[:n]), nil
}
