// SPDX-License-Identifier: MIT

package networkutils

import (
	"fmt"
	"net"
)


func DiscoverInterfaces() ([]InterfaceDetails, error) {
    interfaces, err := net.Interfaces()
    if err != nil {
        return nil, fmt.Errorf("failed to get network interfaces: %w", err)
    }

    var details []InterfaceDetails
    for _, iface := range interfaces {
        if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagBroadcast == 0 {
            continue
        }

        var ips []net.IP
        var subnets []int
        addrs, err := iface.Addrs()
        if err != nil {
            fmt.Printf("Skipping interface %s due to error: %v\n", iface.Name, err)
            continue
        }

        for _, addr := range addrs {
            if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
                ips = append(ips, ipnet.IP)
                mask := ipnet.Mask
                ones, _ := mask.Size()
                subnets = append(subnets, ones)
            }
        }

        if len(ips) > 0 {  // Check if there are any IPs before adding the details
            detail := InterfaceDetails{
                Name:       iface.Name,
                IPs:        ips,
                SubnetBits: subnets,
                MACAddress: iface.HardwareAddr,
            }
            details = append(details, detail)
        }
    }
    return details, nil
}

func GetInterfaceByName(name string) (*InterfaceDetails, error) {
	ifaces, err := DiscoverInterfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range ifaces {
		if iface.Name == name {
			return &iface, nil
		}
	}
	return nil, fmt.Errorf("interface with name '%s' not found", name)
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