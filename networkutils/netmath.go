// SPDX-License-Identifier: MIT

/*
   Shared math functions for network utilities.
*/

package networkutils

import (
	"bytes"
	"math"
	"net"
	"sort"
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

func SortIPs(ips []net.IP) {
    sort.Slice(ips, func(i, j int) bool {
        return bytes.Compare(ips[i], ips[j]) < 0
    })
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
        ips = append(ips, make(net.IP, len(currentIP)))
        copy(ips[len(ips)-1], currentIP)
    }

    return ips
}