// SPDX-License-Identifier: MIT

/*
	Goscan is a simple network scanner that uses ICMP to probe hosts on the network.
*/

package main

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os/user"
	"time"

	"goscan/cliargs"
	"goscan/networkutils"
	"goscan/stats"
	"goscan/webutils"
)

var (
    embeddedFiles embed.FS
    config        cliargs.Config
)

func main() {
    config = cliargs.ParseFlags()

    currentUser, err := user.Current()
    if err != nil || currentUser.Uid != "0" {
        log.Fatal("Application requires administrator privileges to perform network scanning.")
    }

    go stats.MonitorRuntimeStats()

    http.HandleFunc("/", allNetworksHTMLHandler)
    http.HandleFunc("/all-html", allNetworksHTMLHandler)
    http.HandleFunc("/networks", listNetworksHandler)
    http.HandleFunc("/network/", networkHandler)
    http.HandleFunc("/all", allNetworksHandler)
    http.HandleFunc("/stats", statsHandler)

    address := fmt.Sprintf("%s:%s", config.ListenAddress, config.ListenPort)
    log.Printf("Starting server at %s", address)
    log.Fatal(http.ListenAndServe(address, nil))
}

/*
	statsHandler is an HTTP handler that returns current system statistics.
*/
func statsHandler(w http.ResponseWriter, r *http.Request) {
	stats := stats.GetStats()

	response := map[string]interface{}{
		"MemoryAllocKB":   stats.MemAlloc / 1024,
		"SystemMemoryKB":  stats.Sys / 1024,
		"LastGCPauseNs":   stats.LastPauseNs,
		"NumberOfGoroutines": stats.NumGoroutine,
	}

	webutils.WriteJSON(w, response)
}

/*
    listNetworksHandler is an HTTP handler that lists all network interfaces on the system.
*/
func listNetworksHandler(w http.ResponseWriter, r *http.Request) {
	ifaces, err := networkutils.DiscoverInterfaces()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jIfaces := make([]networkutils.InterfaceDetailsJSON, len(ifaces))
	for i, iface := range ifaces {
		jIfaces[i] = iface.ToJSON()
	}

	webutils.WriteJSON(w, jIfaces)
}

/*
	networkHandler is an HTTP handler that probes a specific network interface for active hosts.
*/
func networkHandler(w http.ResponseWriter, r *http.Request) {
    ifaceName := r.URL.Path[len("/network/"):]
    if ifaceName == "" {
        http.Error(w, "Interface name is required.", http.StatusBadRequest)
        return
    }

    iface, err := networkutils.GetInterfaceByName(ifaceName)
    if err != nil {
        http.Error(w, "Interface not found.", http.StatusNotFound)
        return
    }

    activeHosts, err := networkutils.ProbeHostsICMP(iface, config.Timeout)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error probing hosts: %v", err), http.StatusInternalServerError)
        return
    }

    networkutils.SortIPs(activeHosts)
    response := struct {
        InterfaceDetails networkutils.InterfaceDetailsJSON `json:"interface"`
        ActiveHosts      []net.IP                           `json:"activeHosts"`
    }{
        InterfaceDetails: iface.ToJSON(),
        ActiveHosts:      activeHosts,
    }

    webutils.WriteJSON(w, response)
}

/*
	allNetworksHandler is an HTTP handler that probes all network interfaces on the system for active hosts.
	returning a JSON response.
*/
func allNetworksHandler(w http.ResponseWriter, r *http.Request) {
	data, err := networkutils.FetchAllNetworkData(config.Timeout)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-Elapsed-Time", data["elapsedTime"].(time.Duration).String())
	w.Header().Set("X-Total-IPs-Scanned", fmt.Sprintf("%d", data["totalIPsScanned"].(int)))

	webutils.WriteJSON(w, data["results"])
}

/*
    allNetworksHTMLHandler is the same as allNetworksHandler but returns an HTML response.
	Default handler for the root path.
*/
func allNetworksHTMLHandler(w http.ResponseWriter, r *http.Request) {
	data, err := networkutils.FetchAllNetworkData(config.Timeout)
	if err != nil {
		http.Error(w, "Failed to fetch network data", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFS(embeddedFiles, "templates/networks_template.html")
	if err != nil {
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = tmpl.Execute(w, data["results"])
	if err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}