// SPDX-License-Identifier: MIT

/*
   goscan is a simple network scanner that uses ICMP echo requests to
   detect active hosts on a network.

   Usage:
       goscan --interface eth0 --timeout 1000 --measure
       goscan -i eth0 -t 1000 -m

   If the --interface flag is not provided, goscan will scan all
   available interfaces.

   Author: Darius Niminenn
*/

package main

import (
	"flag"
	"fmt"
	"goscan/networkutils"
	"log"
	"sync"
	"time"
)

const (
    colorReset  = "\033[0m"
    colorRed    = "\033[31m"
    colorGreen  = "\033[32m"
    colorYellow = "\033[33m"
    colorBlue   = "\033[34m"
    colorPurple = "\033[35m"
    colorCyan   = "\033[36m"
    colorWhite  = "\033[37m"
    boldText    = "\033[1m"
)

func main() {
    initialTime := time.Now()

    ifaceName := flag.String("interface", "", "Specify the network interface name")
    shortIfaceName := flag.String("i", "", "Specify the network interface name (short)")
    timeout := flag.Int("timeout", 500, "Specify the timeout in milliseconds")
    measureExecutionTime := flag.Bool("measure", false, "Measure the execution time")
    flag.Parse()

    if *ifaceName == "" && *shortIfaceName != "" {
        ifaceName = shortIfaceName
    }

    ifaces, err := networkutils.DiscoverInterfaces()
    if err != nil {
        log.Fatalf("Error discovering interfaces: %v", err)
    }

    var wg sync.WaitGroup
    found := false
    for _, iface := range ifaces {
        if *ifaceName != "" && iface.Name != *ifaceName {
            continue
        }
        found = true
        wg.Add(1)
        go func(iface networkutils.InterfaceDetails) {
            defer wg.Done()
            activeHosts, err := networkutils.ProbeHostsICMP(&iface, time.Duration(*timeout)*time.Millisecond)
            if err != nil {
                fmt.Printf(colorRed+"Error probing hosts on interface %s: %v"+colorReset+"\n", iface.Name, err)
                return
            }
            networkutils.SortIPs(activeHosts)
            if len(activeHosts) > 0 {
                fmt.Printf(boldText+colorCyan+"%s [%s]"+colorReset+"\n", iface.Name, iface.MACAddress)
                for _, host := range activeHosts {
                    fmt.Println(colorGreen+" ✓"+colorReset, host)
                }
                fmt.Println(fmt.Sprintf("Total active hosts on %s: "+boldText+colorGreen+"%d"+colorReset, iface.Name, len(activeHosts)))
            } else {
                fmt.Println("    " + colorPurple + "No active hosts found on this interface." + colorReset)
            }
        }(iface)
    }

    wg.Wait()

    if *ifaceName != "" && !found {
        fmt.Printf(colorRed+"No interface found with the name '%s'"+colorReset+"\n", *ifaceName)
    } else if *measureExecutionTime {
        fmt.Printf(boldText+colorBlue+"Execution time: %v"+colorReset+"\n", time.Since(initialTime))
    }
}
