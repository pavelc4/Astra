package handler

import (
	"net/http"
	"os"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/pavelc4/astra/internal/response"
)

var startTime = time.Now()

var (
	TotalRequests   uint64
	SuccessRequests uint64
	FailedRequests  uint64
)

func HandleHealth(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	var stat syscall.Statfs_t
	var diskTotal, diskFree, diskUsed uint64
	if err := syscall.Statfs("/", &stat); err == nil {
		diskTotal = stat.Blocks * uint64(stat.Bsize)
		diskFree = stat.Bfree * uint64(stat.Bsize)
		diskUsed = diskTotal - diskFree
	}

	data := map[string]any{
		"status":     "ok",
		"version":    runtime.Version(),
		"goroutines": runtime.NumGoroutine(),
		"cpus":       runtime.NumCPU(),
		"uptime":     time.Since(startTime).String(),
		"requests": map[string]any{
			"total":   atomic.LoadUint64(&TotalRequests),
			"success": atomic.LoadUint64(&SuccessRequests),
			"failed":  atomic.LoadUint64(&FailedRequests),
		},
		"cookies": map[string]bool{
			"instagram": os.Getenv("INSTAGRAM_COOKIES") != "",
			"facebook":  os.Getenv("FACEBOOK_COOKIES") != "",
		},
		"disk": map[string]any{
			"total": diskTotal,
			"used":  diskUsed,
			"free":  diskFree,
		},
		"memory": map[string]any{
			"heapAlloc":   m.HeapAlloc,
			"heapInuse":   m.HeapInuse,
			"heapObjects": m.HeapObjects,
			"stackInuse":  m.StackInuse,
			"gcCycles":    m.NumGC,
			"gcPause":     m.PauseTotalNs,
		},
	}

	response.OK(w, data, "Server is healthy")
}
