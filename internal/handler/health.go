package handler

import (
	"net/http"
	"runtime"
	"time"

	"github.com/pavelc4/astra/internal/response"
)

var startTime = time.Now()

func HandleHealth(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	data := map[string]any{
		"status":     "ok",
		"version":    runtime.Version(),
		"goroutines": runtime.NumGoroutine(),
		"cpus":       runtime.NumCPU(),
		"uptime":     time.Since(startTime).String(),
		"memory": map[string]any{
			"heapAlloc":  m.HeapAlloc,
			"heapInuse":  m.HeapInuse,
			"heapObjects": m.HeapObjects,
			"stackInuse": m.StackInuse,
			"gcCycles":   m.NumGC,
			"gcPause":    m.PauseTotalNs,
		},
	}

	response.OK(w, data, "Server is healthy")
}
