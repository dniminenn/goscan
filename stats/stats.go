package stats

import (
	"runtime"
	"sync"
	"time"
)

type Stats struct {
	MemAlloc     uint64
	Sys          uint64
	LastPauseNs  uint64
	NumGoroutine int
}

var (
	currentStats Stats
	statsLock    sync.RWMutex
)

func UpdateStats() {
	statsLock.Lock()
	defer statsLock.Unlock()

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	currentStats.MemAlloc = mem.Alloc
	currentStats.Sys = mem.Sys
	currentStats.LastPauseNs = mem.PauseTotalNs
	currentStats.NumGoroutine = runtime.NumGoroutine()
}

func GetStats() Stats {
	statsLock.RLock()
	defer statsLock.RUnlock()
	return currentStats
}

func MonitorRuntimeStats() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		UpdateStats()
	}
}
