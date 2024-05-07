package stats

import (
	"fmt"
	"runtime"
	"time"
)

func MonitorRuntimeStats() {
	var m runtime.MemStats
	var lastPauseNs uint64

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			runtime.ReadMemStats(&m)
			total := m.Alloc
			sys := m.Sys
			pause := m.PauseTotalNs - lastPauseNs

            /* Mem Alloc is the total number of bytes allocated and not yet freed.
                Sys is the total number of bytes obtained from the OS.
                Pause is the total time spent in GC pauses. */
			fmt.Printf("\r\x1b[2KMemory Alloc: %d KB, Sys: %d KB, Pause: %d ns",
				total/1024, sys/1024, pause)

			lastPauseNs = m.PauseTotalNs
		}
	}
}