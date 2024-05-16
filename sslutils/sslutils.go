package sslutils

import (
	"crypto/tls"
	"log"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// CertReloader is a struct that manages the reloading of SSL certificates.
type CertReloader struct {
    mu       sync.RWMutex
    cert     *tls.Certificate
    certFile string
    keyFile  string
}

// NewCertReloader creates a new CertReloader instance and starts watching for changes in the certificate files.
func NewCertReloader(certFile, keyFile string) (*CertReloader, error) {
    reloader := &CertReloader{certFile: certFile, keyFile: keyFile}
    err := reloader.loadCert()
    if err != nil {
        return nil, err
    }
    go reloader.watchCert()
    return reloader, nil
}

// loadCert loads the SSL certificate from the files specified in the CertReloader instance.
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

// watchCert watches for changes in the certificate files and reloads the certificate when necessary.
func (r *CertReloader) watchCert() {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        log.Println(err)
        return
    }
    defer watcher.Close()

    err = watcher.Add(r.certFile)
    if err != nil {
        log.Println(err)
        return
    }
    err = watcher.Add(r.keyFile)
    if err != nil {
        log.Println(err)
        return
    }

    var certModified, keyModified bool
    for {
        select {
        case event, ok := <-watcher.Events:
            if !ok {
                return
            }
            if event.Op&fsnotify.Write == fsnotify.Write {
                if event.Name == r.certFile {
                    certModified = true
                } else if event.Name == r.keyFile {
                    keyModified = true
                }
                if certModified && keyModified {
                    time.Sleep(1 * time.Second) // Sleep for a second to ensure both files are fully written
                    err := r.loadCert()
                    if err != nil {
                        log.Println("Failed to reload certificate:", err)
                    } else {
                        log.Println("Certificate reloaded successfully.")
                    }
                    certModified, keyModified = false, false
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

// GetCertificateFunc returns a function that can be used to get the SSL certificate.
func (r *CertReloader) GetCertificateFunc() func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
    return func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
        r.mu.RLock()
        defer r.mu.RUnlock()
        return r.cert, nil
    }
}