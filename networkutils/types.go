// SPDX-License-Identifier: MIT

package networkutils

import (
	"net"
)

type InterfaceDetails struct {
    Name       string
    IPs        []net.IP
    SubnetBits []int
    MACAddress net.HardwareAddr
}

type InterfaceDetailsJSON struct {
	InterfaceDetails
	MACAddress string `json:"MACAddress"`
}

type pingResult struct {
	ip  net.IP
	err error
}

func (iface *InterfaceDetails) ToJSON() InterfaceDetailsJSON {
	return InterfaceDetailsJSON{
		InterfaceDetails: *iface,
		MACAddress:       iface.MACAddress.String(),
	}
}