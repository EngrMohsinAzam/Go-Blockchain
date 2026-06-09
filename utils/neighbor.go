package utils

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// IsFoundHost returns true if the host:port is reachable via TCP.
func IsFoundHost(host string, port uint16) bool {
	target := net.JoinHostPort(host, strconv.Itoa(int(port)))
	conn, err := net.DialTimeout("tcp", target, 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// FindNeighbors discovers other nodes on the local network by scanning IP range and ports.
// myHost e.g. "127.0.0.1", startIp/endIp are offsets for the last octet (0-3), startPort/endPort are port range.
func FindNeighbors(myHost string, myPort uint16, startIp uint8, endIp uint8, startPort uint16, endPort uint16) []string {
	address := net.JoinHostPort(myHost, strconv.Itoa(int(myPort)))
	parts := strings.Split(myHost, ".")
	if len(parts) != 4 {
		return nil
	}
	lastOctet, err := strconv.Atoi(parts[3])
	if err != nil {
		return nil
	}
	prefix := strings.Join(parts[:3], ".") + "."
	neighbors := make([]string, 0)

	for port := startPort; port <= endPort; port++ {
		for ip := startIp; ip <= endIp; ip++ {
			guessHost := fmt.Sprintf("%s%d", prefix, lastOctet+int(ip))
			guessTarget := net.JoinHostPort(guessHost, strconv.Itoa(int(port)))
			if guessTarget != address && IsFoundHost(guessHost, port) {
				neighbors = append(neighbors, guessTarget)
			}
		}
	}
	return neighbors
}

// GetHost returns the local host IP for neighbor discovery (127.0.0.1 for local dev).
func GetHost() string {
	host, err := os.Hostname()
	if err != nil {
		return "127.0.0.1"
	}
	addrs, err := net.LookupIP(host)
	if err != nil || len(addrs) == 0 {
		return "127.0.0.1"
	}
	for _, a := range addrs {
		if a.To4() != nil {
			return a.String()
		}
	}
	return "127.0.0.1"
}
