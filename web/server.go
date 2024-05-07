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

	"goscan/networkutils"
	"goscan/stats"
	"goscan/webutils"
)

// Embedding the templates directory
//go:embed templates/*
var embeddedFiles embed.FS

func main() {
	// Check if the user is running as root (admin privileges) to use ICMP/ping
	currentUser, err := user.Current()
	if err != nil || currentUser.Uid != "0" {
		log.Fatal("Application requires administrator privileges to perform network scanning.")
	}

	// Initialize runtime statistics monitoring
	go stats.MonitorRuntimeStats()

    http.HandleFunc("/all-html", allNetworksHTMLHandler)
	http.HandleFunc("/networks", listNetworksHandler)
	http.HandleFunc("/network/", networkHandler)
	http.HandleFunc("/all", allNetworksHandler)

	log.Println("Starting server at port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}


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

	activeHosts, err := networkutils.ProbeHostsICMP(iface, time.Millisecond*100)
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

func allNetworksHandler(w http.ResponseWriter, r *http.Request) {
	data, err := networkutils.FetchAllNetworkData()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-Elapsed-Time", data["elapsedTime"].(time.Duration).String())
	w.Header().Set("X-Total-IPs-Scanned", fmt.Sprintf("%d", data["totalIPsScanned"].(int)))

	webutils.WriteJSON(w, data["results"])
}

func allNetworksHTMLHandler(w http.ResponseWriter, r *http.Request) {
	data, err := networkutils.FetchAllNetworkData()
	if err != nil {
		http.Error(w, "Failed to fetch network data", http.StatusInternalServerError)
		return
	}

	// Use embedded file system
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