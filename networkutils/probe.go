// SPDX-License-Identifier: MIT

/*
   ICMP probing logic.
   Uses the go-ping library to send ICMP echo requests to a range of IP addresses.
*/

package networkutils

import (
	"net"
	"sync"
	"time"

	"github.com/go-ping/ping"
)

type pingResult struct {
	ip  net.IP
	err error
}

func pingHost(ip net.IP, timeout time.Duration, resultsChan chan<- pingResult, wg *sync.WaitGroup) {
	defer wg.Done()
	var retryCount int
	maxRetries := 3
	for retryCount < maxRetries {
		pinger, err := ping.NewPinger(ip.String())
		if err != nil {
			resultsChan <- pingResult{nil, err}
			return
		}
		pinger.SetPrivileged(true)
		pinger.Count = 3
		pinger.Timeout = timeout
		var received bool
		pinger.OnRecv = func(pkt *ping.Packet) {
			received = true
			resultsChan <- pingResult{pkt.IPAddr.IP, nil}
		}
		pinger.Run()
		if received {
			return
		}
		retryCount++
		timeout *= 2
	}
	resultsChan <- pingResult{nil, nil}
}

func handleResults(resultsChan <-chan pingResult, activeHosts *[]net.IP, returnError *error, mu *sync.Mutex) {
	for result := range resultsChan {
		if result.err != nil && *returnError == nil {
			*returnError = result.err
			continue
		}
		if result.ip != nil {
			mu.Lock()
			*activeHosts = append(*activeHosts, result.ip)
			mu.Unlock()
		}
	}
}

func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] != 0 {
			break
		}
	}
}

func generateIPs(ip net.IP, subnetBits int) []net.IP {
	var ips []net.IP

	network := ip.Mask(net.CIDRMask(subnetBits, 32))
	broadcast := make(net.IP, len(network))

	for i, b := range network {
		broadcast[i] = b | ^net.CIDRMask(subnetBits, 32)[i]
	}

	currentIP := make(net.IP, len(network))
	copy(currentIP, network)
	for ; !currentIP.Equal(broadcast); incrementIP(currentIP) {
		ips = append(ips, append(net.IP(nil), currentIP...))
	}

	return ips
}

func ProbeHostsICMP(ifaceDetails *InterfaceDetails, initialTimeout time.Duration) ([]net.IP, error) {
	var wg sync.WaitGroup
	resultsChan := make(chan pingResult, 1024)
	var activeHosts []net.IP
	var returnError error
	var mu sync.Mutex

	sem := make(chan struct{}, 1024)
	go handleResults(resultsChan, &activeHosts, &returnError, &mu)

	for i, ip := range ifaceDetails.IPs {
		subnetBits := ifaceDetails.SubnetBits[i]
		allIPs := generateIPs(ip, subnetBits)
		for _, ip := range allIPs {
			wg.Add(1)
			sem <- struct{}{}
			go func(ip net.IP) {
				defer func() { <-sem }()
				pingHost(ip, initialTimeout, resultsChan, &wg)
			}(ip)
		}
	}

	wg.Wait()
	close(resultsChan)

	return activeHosts, returnError
}
