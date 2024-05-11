// SPDX-License-Identifier: MIT

/*
	Goscan is a simple network scanner that uses ICMP to probe hosts on the network.
*/

package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"os/user"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"

	"goscan/cliargs"
	"goscan/networkutils"
	"goscan/stats"
)

var (
	//go:embed templates/*
	embeddedFiles embed.FS
	config cliargs.Config
)

func main() {
	config = cliargs.ParseFlags()

	currentUser, err := user.Current()
	if err != nil || currentUser.Uid != "0" {
		log.Fatal("Application requires administrator privileges to perform network scanning.")
	}

	go stats.MonitorRuntimeStats()

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	router.Use(gin.Logger(), gin.Recovery())

	router.GET("/", allNetworksHTMLHandler)
	router.GET("/networks", listNetworksHandler)
	router.GET("/network/:iface", networkHandler)
	router.GET("/all", allNetworksHandler)
	router.GET("/stats", statsHandler)

	address := fmt.Sprintf("%s:%s", config.ListenAddress, config.ListenPort)
	log.Printf("Starting server at %s", address)
	log.Fatal(router.Run(address))
}

func statsHandler(c *gin.Context) {
	stats := stats.GetStats()

	c.JSON(http.StatusOK, gin.H{
		"MemoryAllocKB":      stats.MemAlloc / 1024,
		"SystemMemoryKB":     stats.Sys / 1024,
		"LastGCPauseNs":      stats.LastPauseNs,
		"NumberOfGoroutines": stats.NumGoroutine,
	})
}

func listNetworksHandler(c *gin.Context) {
	ifaces, err := networkutils.DiscoverInterfaces()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ifaces)
}

func networkHandler(c *gin.Context) {
	ifaceName := c.Param("iface")
	if ifaceName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Interface name is required."})
		return
	}

	iface, err := networkutils.GetInterfaceByName(ifaceName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Interface not found."})
		return
	}

	activeHosts, err := networkutils.ProbeHostsICMP(iface, config.Timeout)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error probing hosts: %v", err)})
		return
	}

	networkutils.SortIPs(activeHosts)
	c.JSON(http.StatusOK, gin.H{
		"interface":    iface.ToJSON(),
		"activeHosts":  activeHosts,
	})
}

func allNetworksHandler(c *gin.Context) {
	data, err := networkutils.FetchAllNetworkData(config.Timeout)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("X-Elapsed-Time", data["elapsedTime"].(time.Duration).String())
	c.Header("X-Total-IPs-Scanned", fmt.Sprintf("%d", data["totalIPsScanned"].(int)))
	c.JSON(http.StatusOK, data["results"])
}

func allNetworksHTMLHandler(c *gin.Context) {
	tmpl, err := template.ParseFS(embeddedFiles, "templates/networks_template.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load template")
		return
	}

	c.Header("Content-Type", "text/html")
	err = tmpl.Execute(c.Writer, nil) // No need to pass data to the template
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to render template")
		return
	}
}