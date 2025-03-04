// SPDX-License-Identifier: MIT

/*
   Host discovery logic using multiple techniques:
   - ARP for local network discovery (fastest)
   - ICMP echo requests
   - TCP port scans for common services
*/

package networkutils

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/go-ping/ping"
	"github.com/j-keck/arping"
)

const (
	maxConcurrentScans = 1024
	maxRetries         = 2
)

var commonPorts = []int{
	// Web services (very common)
	80, 443, 8080, 8443, 8000, 8888,
	// Remote access
	22, 23, 3389, 5900,
	// File sharing
	445, 139, 21,
	// Database
	1433, 3306, 5432, 6379, 27017,
	// Mail
	25, 110, 143, 587, 993, 995,
	// DNS/DHCP
	53, 67, 68,
	// Other common services
	123, 161, 500, 1723, 5060, 8443, 9100,
}

type hostResult struct {
	ip     net.IP
	active bool
	err    error
}

// incrementIP increments an IP address by 1
func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] != 0 {
			break
		}
	}
}

// generateIPs generates all IPs in a subnet
func generateIPs(ip net.IP, subnetBits int) []net.IP {
	var ips []net.IP

	network := ip.Mask(net.CIDRMask(subnetBits, 32))
	broadcast := make(net.IP, len(network))

	for i, b := range network {
		broadcast[i] = b | ^net.CIDRMask(subnetBits, 32)[i]
	}

	currentIP := make(net.IP, len(network))
	copy(currentIP, network)
	incrementIP(currentIP)

	for ; !currentIP.Equal(broadcast); incrementIP(currentIP) {
		ipCopy := make(net.IP, len(currentIP))
		copy(ipCopy, currentIP)
		ips = append(ips, ipCopy)
	}

	return ips
}

// isLocalNetwork checks if the subnet is a local network
func isLocalNetwork(subnetBits int) bool {
	// Usually /24 or smaller is local network
	return subnetBits >= 24
}

// arpScan attempts to discover hosts using ARP
func arpScan(ip net.IP, timeout time.Duration) bool {
	arping.SetTimeout(timeout)

	_, _, err := arping.Ping(ip)
	return err == nil
}

// icmpScan attempts to discover hosts using ICMP echo requests
func icmpScan(ip net.IP, timeout time.Duration) bool {
	var retryCount int
	currentTimeout := timeout

	for retryCount < maxRetries {
		pinger, err := ping.NewPinger(ip.String())
		if err != nil {
			return false
		}

		pinger.SetPrivileged(true)
		pinger.Count = 2
		pinger.Interval = 50 * time.Millisecond
		pinger.Timeout = currentTimeout
		pinger.Size = 56

		jitter := time.Duration(rand.Intn(100)) * time.Millisecond
		time.Sleep(jitter)

		err = pinger.Run()
		stats := pinger.Statistics()

		if err == nil && stats.PacketsRecv > 0 {
			return true
		}

		retryCount++
		currentTimeout *= 2
	}

	return false
}

// tcpScan attempts to discover hosts by checking for common open TCP ports
func tcpScan(ip net.IP, timeout time.Duration) bool {
	for _, port := range commonPorts {
		var address string
		if ip.To4() == nil {
			// IPv6 address
			address = fmt.Sprintf("[%s]:%d", ip.String(), port)
		} else {
			// IPv4 address
			address = fmt.Sprintf("%s:%d", ip.String(), port)
		}

		conn, err := net.DialTimeout("tcp", address, timeout/2)
		if err == nil {
			conn.Close()
			return true
		}
	}
	return false
}

// probeHost attempts to discover if a host is active using multiple methods
func probeHost(ip net.IP, timeout time.Duration, isLocal bool, resultsChan chan<- hostResult, wg *sync.WaitGroup) {
	defer wg.Done()

	// Try ARP first if it's a local network (fastest method)
	if isLocal && arpScan(ip, timeout/2) {
		resultsChan <- hostResult{ip: ip, active: true}
		return
	}

	if icmpScan(ip, timeout) {
		resultsChan <- hostResult{ip: ip, active: true}
		return
	}

	// if tcpScan(ip, timeout) {
	// 	resultsChan <- hostResult{ip: ip, active: true}
	// 	return
	// }

	resultsChan <- hostResult{ip: ip, active: false}
}

// handleResults collects the results of host probing
func handleResults(resultsChan <-chan hostResult, activeHosts *[]net.IP, returnError *error, mu *sync.Mutex) {
	for result := range resultsChan {
		if result.err != nil && *returnError == nil {
			*returnError = result.err
			continue
		}
		if result.active {
			mu.Lock()
			*activeHosts = append(*activeHosts, result.ip)
			mu.Unlock()
		}
	}
}

// ProbeHosts probes hosts on a network interface using multiple methods
func ProbeHosts(ifaceDetails *InterfaceDetails, initialTimeout time.Duration) ([]net.IP, []net.IP, error) {
	var wg sync.WaitGroup
	resultsChan := make(chan hostResult, maxConcurrentScans)
	var activeHosts []net.IP
	var allHosts []net.IP // Added to track all scanned hosts
	var returnError error
	var mu sync.Mutex

	sem := make(chan struct{}, maxConcurrentScans)

	go handleResults(resultsChan, &activeHosts, &returnError, &mu)

	for i, ip := range ifaceDetails.IPs {
		subnetBits := ifaceDetails.SubnetBits[i]
		ipList := generateIPs(ip, subnetBits)
		isLocal := isLocalNetwork(subnetBits)

		// Store all IPs being scanned
		mu.Lock()
		allHosts = append(allHosts, ipList...)
		mu.Unlock()

		for _, targetIP := range ipList {
			wg.Add(1)
			sem <- struct{}{}

			go func(ip net.IP, isLocal bool) {
				defer func() { <-sem }()
				probeHost(ip, initialTimeout, isLocal, resultsChan, &wg)
			}(targetIP, isLocal)
		}
	}

	wg.Wait()
	close(resultsChan)

	return activeHosts, allHosts, returnError
}
