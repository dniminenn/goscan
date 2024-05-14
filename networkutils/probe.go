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

func pingHost(ip net.IP, timeout time.Duration, resultsChan chan<- pingResult) {
	var (
		retryCount int
		maxRetries = 2
		pinger     *ping.Pinger
		err        error
	)
	for {
		pinger, err = ping.NewPinger(ip.String())
		if err != nil {
			resultsChan <- pingResult{nil, err}
			return
		}
		pinger.SetPrivileged(true)
		pinger.Size = 56
		pinger.Count = 3
		pinger.Timeout = timeout
		pinger.OnRecv = func(pkt *ping.Packet) {
			resultsChan <- pingResult{pkt.IPAddr.IP, nil}
		}
		pinger.Run()
		if pinger.Statistics().PacketsRecv > 0 {
			break
		}
		retryCount++
		if retryCount > maxRetries {
			break
		}
		timeout *= 2
	}
}

func handleResults(resultsChan <-chan pingResult, activeHosts *[]net.IP, returnError *error) {
	for result := range resultsChan {
		if result.err != nil && *returnError == nil {
			*returnError = result.err
			continue
		}
		if result.ip != nil {
			*activeHosts = append(*activeHosts, result.ip)
		}
	}
}

func ProbeHostsICMP(ifaceDetails *InterfaceDetails, initialTimeout time.Duration) ([]net.IP, error) {
	var wg sync.WaitGroup
	resultsChan := make(chan pingResult, 1024)
	var activeHosts []net.IP
	var returnError error

	sem := make(chan struct{}, 1024)

	go handleResults(resultsChan, &activeHosts, &returnError)

	for i, ip := range ifaceDetails.IPs {
		subnetBits := ifaceDetails.SubnetBits[i]
		allIPs := generateIPs(ip, subnetBits)
		for _, ip := range allIPs {
			wg.Add(1)
			sem <- struct{}{}
			go func(ip net.IP) {
				defer wg.Done()
				defer func() { <-sem }()
				pingHost(ip, initialTimeout, resultsChan)
			}(ip)
		}
	}

	wg.Wait()
	close(resultsChan)

	return activeHosts, returnError
}

