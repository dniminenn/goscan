// SPDX-License-Identifier: MIT

package networkutils

import (
	"math"
	"net"
)

func CalcSubnetSize(subnetBits []int) int {
	totalIPs := 0
	for _, bits := range subnetBits {
		if bits > 0 && bits <= 32 {
			// Calculate the number of IPs in this subnet.
			// 32 - bits gives the number of hosts bits in the subnet mask.
			// For example, a /24 subnet has 32 - 24 = 8 bits for hosts, which means 2^8 - 2 addresses (excluding network and broadcast).
			totalIPs += int(math.Pow(2, float64(32-bits))) - 2
		}
	}
	return totalIPs
}

func CalculateTotalIPsScanned(ifaces []InterfaceDetails) int {
	totalIPsScanned := 0
	for _, iface := range ifaces {
		totalIPsScanned += CalcSubnetSize(iface.SubnetBits)
	}
	return totalIPsScanned
}

func incrementIP(ip net.IP) {
    for j := len(ip) - 1; j >= 0; j-- {
        ip[j]++
        if ip[j] != 0 {
            break
        }
    }
}