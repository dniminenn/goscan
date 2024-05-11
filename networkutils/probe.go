// SPDX-License-Identifier: MIT

/*
   ICMP probing logic.
   Uses the go-ping library to send ICMP echo requests to a range of IP addresses.
*/

package networkutils

import (
	"bytes"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/go-ping/ping"
)

func SortIPs(ips []net.IP) {
    sort.Slice(ips, func(i, j int) bool {
        return bytes.Compare(ips[i], ips[j]) < 0
    })
}

func ProbeHostsICMP(ifaceDetails *InterfaceDetails, initialTimeout time.Duration) ([]net.IP, error) {
	var wg sync.WaitGroup
	resultsChan := make(chan pingResult, 100)
	var activeHosts []net.IP
	var returnError error

	sem := make(chan struct{}, 2048)

	go func() {
		for result := range resultsChan {
			if result.err != nil && returnError == nil {
				returnError = result.err
				continue
			}
			if result.ip != nil {
				activeHosts = append(activeHosts, result.ip)
			}
		}
	}()

	for i, ip := range ifaceDetails.IPs {
		subnetBits := ifaceDetails.SubnetBits[i]
		allIPs := generateIPs(ip, subnetBits)
		for _, ip := range allIPs {
			wg.Add(1)
			sem <- struct{}{}
			go func(ip net.IP, timeout time.Duration) {
				defer wg.Done()
				defer func() { <-sem }()
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
					pinger.Size = 32
					pinger.Count = 5
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
			}(ip, initialTimeout)
		}
	}

	wg.Wait()
	close(resultsChan)

	return activeHosts, returnError
}

