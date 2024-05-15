// SPDX-License-Identifier: MIT

/*
	Goscan is a simple network scanner that uses ICMP to probe hosts on the network.
*/

package main

import (
	"crypto/tls"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os/user"
	"sync"
	"text/template"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"

	"goscan/config"
	"goscan/networkutils"
	"goscan/stats"
)

var (
	//go:embed templates/*
	embeddedFiles embed.FS
)

func main() {
	config.ParseFlagsServer()
	serverConfig := config.GetServerConfig()

	currentUser, err := user.Current()
	if err != nil || currentUser.Uid != "0" {
		log.Fatal("Application requires administrator privileges to perform network scanning.")
	}

	go stats.MonitorRuntimeStats()

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	templatesFS, err := fs.Sub(embeddedFiles, "templates/static")
	if err != nil {
		log.Fatal(err)
	}
	router.StaticFS("/static", http.FS(templatesFS))

	router.GET("/", allNetworksHTMLHandler)
	router.GET("/networks", listNetworksHandler)
	router.GET("/network/:iface", networkHandler)
	router.GET("/all", allNetworksHandler)
	router.GET("/stats", statsHandler)

	address := fmt.Sprintf("%s:%s", serverConfig.ListenAddress, serverConfig.ListenPort)
	log.Printf("Starting server at %s", address)

	if serverConfig.SSLCertFile != "" && serverConfig.SSLKeyFile != "" {
		reloader, err := NewCertReloader(serverConfig.SSLCertFile, serverConfig.SSLKeyFile)
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
	tmpl, err := template.ParseFS(embeddedFiles, "templates/allnetworks.html")
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

// SSL certificate reloader
type CertReloader struct {
	mu       sync.RWMutex
	cert     *tls.Certificate
	certFile string
	keyFile  string
}

func NewCertReloader(certFile, keyFile string) (*CertReloader, error) {
	reloader := &CertReloader{certFile: certFile, keyFile: keyFile}
	err := reloader.loadCert()
	if err != nil {
		return nil, err
	}
	go reloader.watchCert()
	return reloader, nil
}

func (r *CertReloader) loadCert() error {
	cert, err := tls.LoadX509KeyPair(r.certFile, r.keyFile)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.cert = &cert
	r.mu.Unlock()
	return nil
}

func (r *CertReloader) watchCert() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	err = watcher.Add(r.certFile)
	if err != nil {
		log.Fatal(err)
	}
	err = watcher.Add(r.keyFile)
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Println("Detected change in certificate files, reloading...")
				time.Sleep(1 * time.Second) // Sleep for a second to ensure the file is fully written
				err := r.loadCert()
				if err != nil {
					log.Println("Failed to reload certificate:", err)
				} else {
					log.Println("Certificate reloaded successfully.")
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("Error watching certificate files:", err)
		}
	}
}

func (r *CertReloader) GetCertificateFunc() func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
		r.mu.RLock()
		defer r.mu.RUnlock()
		return r.cert, nil
	}
}
