package main

import (
	"fmt"
	"goscan/config"
	"goscan/networkutils"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
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

func runCLI(cmd *cobra.Command, args []string) {
	initialTime := time.Now()
	ifaceName, _ := cmd.Flags().GetString("interface")
	timeout, _ := cmd.Flags().GetInt("timeout")
	measureExecutionTime, _ := cmd.Flags().GetBool("measure")
	showMode, _ := cmd.Flags().GetString("show")
	scriptable, _ := cmd.Flags().GetBool("scriptable")

	showMode = strings.ToLower(showMode)

	switch showMode {
	case "online", "used", "taken":
		showMode = "alive"
	case "offline", "unused", "free":
		showMode = "available"
	}

	if showMode == "alive" || showMode == "available" {
		scriptable = true
	}

	cfg := config.GetServerConfig()
	cfg.Timeout = time.Duration(timeout) * time.Millisecond
	config.SetServerConfig(cfg)

	ifaces, err := networkutils.DiscoverInterfaces()
	if err != nil {
		log.Fatalf("Error discovering interfaces: %v", err)
	}

	var wg sync.WaitGroup
	found := false

	for _, iface := range ifaces {
		if ifaceName != "" && iface.Name != ifaceName {
			continue
		}
		found = true
		wg.Add(1)
		go func(iface networkutils.InterfaceDetails) {
			defer wg.Done()
			activeHosts, allHosts, err := networkutils.ProbeHosts(&iface, time.Duration(timeout)*time.Millisecond)
			if err != nil {
				fmt.Printf(colorRed+"Error probing hosts on interface %s: %v"+colorReset+"\n", iface.Name, err)
				return
			}

			networkutils.SortIPs(activeHosts)

			if len(allHosts) > 0 {
				// Create a map of inactive hosts (all hosts - active hosts)
				inactiveMap := make(map[string]bool)
				for _, host := range allHosts {
					inactiveMap[host.String()] = true
				}

				for _, host := range activeHosts {
					delete(inactiveMap, host.String()) // Remove active hosts from the map
				}

				// Extract inactive hosts from the map
				var inactiveHosts []net.IP
				for hostStr := range inactiveMap {
					ip := net.ParseIP(hostStr)
					if ip != nil {
						inactiveHosts = append(inactiveHosts, ip)
					}
				}
				networkutils.SortIPs(inactiveHosts)

				// For scriptable mode, just print the IPs without any formatting
				if scriptable {
					switch showMode {
					case "alive":
						for _, host := range activeHosts {
							fmt.Println(host.String())
						}
					case "available":
						for _, host := range inactiveHosts {
							fmt.Println(host.String())
						}
					}
					return
				}

				fmt.Printf(boldText+colorCyan+"Interface: %s [%s]"+colorReset+"\n", iface.Name, iface.MACAddress)

				table := tablewriter.NewWriter(os.Stdout)

				switch showMode {
				case "all":
					table.SetHeader([]string{"Active Hosts", "Available IPs"})
					table.SetHeaderColor(
						tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor},
						tablewriter.Colors{tablewriter.Bold, tablewriter.FgBlueColor},
					)
				case "alive":
					table.SetHeader([]string{"Active Hosts", ""})
					table.SetHeaderColor(
						tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor},
						tablewriter.Colors{tablewriter.Bold},
					)
				case "available":
					table.SetHeader([]string{"Available IPs", ""})
					table.SetHeaderColor(
						tablewriter.Colors{tablewriter.Bold, tablewriter.FgBlueColor},
						tablewriter.Colors{tablewriter.Bold},
					)
				}

				table.SetAlignment(tablewriter.ALIGN_LEFT)
				table.SetBorder(false)
				table.SetColumnSeparator("   ")
				table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
				table.SetAutoWrapText(false)

				activeCount := len(activeHosts)
				inactiveCount := len(inactiveHosts)
				totalCount := activeCount + inactiveCount

				switch showMode {
				case "all":
					table.Append([]string{
						fmt.Sprintf("%s%d hosts online%s", colorGreen, activeCount, colorReset),
						fmt.Sprintf("%s%d IPs available%s", colorBlue, inactiveCount, colorReset),
					})
					table.Append([]string{"----------------", "----------------"})
				case "alive":
					table.Append([]string{
						fmt.Sprintf("%s%d hosts online%s", colorGreen, activeCount, colorReset),
						"",
					})
					table.Append([]string{"----------------", ""})
				case "available":
					table.Append([]string{
						fmt.Sprintf("%s%d IPs available%s", colorBlue, inactiveCount, colorReset),
						"",
					})
					table.Append([]string{"----------------", ""})
				}

				switch showMode {
				case "all":
					maxRows := activeCount
					if inactiveCount > maxRows {
						maxRows = inactiveCount
					}

					for i := 0; i < maxRows; i++ {
						row := []string{"", ""}

						if i < activeCount {
							row[0] = activeHosts[i].String()
						}

						if i < inactiveCount {
							row[1] = inactiveHosts[i].String()
						}

						table.Append(row)
					}
				case "alive":
					for _, host := range activeHosts {
						table.Append([]string{host.String(), ""})
					}
				case "available":
					for _, host := range inactiveHosts {
						table.Append([]string{host.String(), ""})
					}
				}

				table.Render()

				fmt.Printf("\nTotal IPs in subnet: %s%d%s\n", boldText, totalCount, colorReset)
				fmt.Printf("Hosts responding: %s%s%d%s (%0.1f%%)\n",
					boldText, colorGreen, activeCount, colorReset,
					float64(activeCount)/float64(totalCount)*100)
			} else {
				if !scriptable {
					fmt.Println("    " + colorPurple + "No hosts found on this interface." + colorReset)
				}
			}
		}(iface)
	}

	wg.Wait()

	if ifaceName != "" && !found && !scriptable {
		fmt.Printf(colorRed+"No interface found with the name '%s'"+colorReset+"\n", ifaceName)
	} else if measureExecutionTime && !scriptable {
		fmt.Printf(boldText+colorBlue+"Execution time: %v"+colorReset+"\n", time.Since(initialTime))
	}
}
