// SPDX-License-Identifier: MIT

/*
   Network data logic.
*/

package networkutils

import (
	"sync"
	"time"
)

func calculateTotalIPsScanned(ifaces []InterfaceDetails) int {
	totalIPsScanned := 0
	for _, iface := range ifaces {
		totalIPsScanned += CalcSubnetSize(iface.SubnetBits)
	}
	return totalIPsScanned
}

func FetchAllNetworkData(timeout time.Duration) (map[string]interface{}, error) {
	startTime := time.Now()
	ifaces, err := DiscoverInterfaces()
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	results := make(map[string]interface{})
	var mu sync.Mutex

	for _, iface := range ifaces {
		wg.Add(1)
		go func(iface InterfaceDetails) {
			defer wg.Done()
			activeHosts, err := ProbeHostsICMP(&iface, timeout)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				results[iface.Name] = map[string]interface{}{"error": err.Error()}
				return
			}
			SortIPs(activeHosts)
			totalIpsScanned := CalcSubnetSize(iface.SubnetBits)

			results[iface.Name] = map[string]interface{}{
				"MACAddress":      iface.MACAddress.String(),
				"TotalIPsScanned": totalIpsScanned,
				"activeHosts":     activeHosts,
			}
		}(iface)
	}
	wg.Wait()

	elapsed := time.Since(startTime)

	return map[string]interface{}{
		"results":         results,
		"elapsedTime":     elapsed,
		"totalIPsScanned": calculateTotalIPsScanned(ifaces),
	}, nil
}
