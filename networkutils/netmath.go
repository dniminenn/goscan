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
		totalIPs += CalcSubnetSizeSingle(bits)
	}
	return totalIPs
}

func CalcSubnetSizeSingle(bits int) int {
	if bits > 0 && bits <= 32 {
		return int(math.Pow(2, float64(32-bits))) - 2
	}
	return 0
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
