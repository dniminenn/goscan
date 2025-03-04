package main

import (
	"crypto/tls"
	"fmt"
	"goscan/cmd/assets"
	"goscan/config"
	"goscan/networkutils"
	"goscan/sslutils"
	"goscan/stats"
	"io/fs"
	"log"
	"net/http"
	"os/user"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

func runServer(cmd *cobra.Command, args []string) {
	listenAddress, _ := cmd.Flags().GetString("listen-address")
	listenPort, _ := cmd.Flags().GetString("listen-port")
	timeout, _ := cmd.Flags().GetInt("timeout")
	sslCert, _ := cmd.Flags().GetString("ssl-cert")
	sslKey, _ := cmd.Flags().GetString("ssl-key")
	maxSubnetSize, _ := cmd.Flags().GetInt("max-subnet-size")

	cfg := config.GetServerConfig()
	cfg.ListenAddress = listenAddress
	cfg.ListenPort = listenPort
	cfg.Timeout = time.Duration(timeout) * time.Millisecond
	cfg.SSLCertFile = sslCert
	cfg.SSLKeyFile = sslKey
	cfg.MaxSubnetSize = maxSubnetSize
	config.SetServerConfig(cfg)

	currentUser, err := user.Current()
	if err != nil || currentUser.Uid != "0" {
		log.Fatal("Application requires administrator privileges to perform network scanning.")
	}

	go stats.MonitorRuntimeStats()

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	staticFS, err := fs.Sub(assets.Templates, "templates/static")
	if err != nil {
		log.Fatal(err)
	}
	router.StaticFS("/static", http.FS(staticFS))

	router.GET("/", allNetworksHTMLHandler)
	router.GET("/networks", listNetworksHandler)
	router.GET("/network/:iface", networkHandler)
	router.GET("/all", allNetworksHandler)
	router.GET("/stats", statsHandler)

	address := fmt.Sprintf("%s:%s", listenAddress, listenPort)
	log.Printf("Starting server at %s", address)

	if sslCert != "" && sslKey != "" {
		reloader, err := sslutils.NewCertReloader(sslCert, sslKey)
		if err != nil {
			log.Fatalf("Failed to initialize certificate reloader: %v", err)
		}

		srv := &http.Server{
			Addr:    address,
			Handler: router,
			TLSConfig: &tls.Config{
				GetCertificate: reloader.GetCertificateFunc(),
			},
		}

		err = srv.ListenAndServeTLS("", "")
		if err != nil {
			log.Fatalf("Failed to run server with TLS: %v", err)
		}
	} else {
		err := router.Run(address)
		if err != nil {
			log.Fatalf("Failed to run server: %v", err)
		}
	}
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

	config := config.GetServerConfig()
	activeHosts, allHosts, err := networkutils.ProbeHosts(iface, config.Timeout)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error probing hosts: %v", err)})
		return
	}

	networkutils.SortIPs(activeHosts)
	c.JSON(http.StatusOK, gin.H{
		"interface":   iface.ToJSON(),
		"activeHosts": activeHosts,
		"totalHosts":  len(allHosts),
	})
}

func allNetworksHandler(c *gin.Context) {
	config := config.GetServerConfig()
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
	tmpl, err := template.ParseFS(assets.Templates, "templates/allnetworks.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load template")
		return
	}

	c.Header("Content-Type", "text/html")
	err = tmpl.Execute(c.Writer, nil)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to render template")
		return
	}
}
