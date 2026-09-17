package services

import (
	"runtime"
	"time"
)

var appStartTime = time.Now().UTC()

// HostMetrics описывает метрики среды выполнения и ресурсов сервера.
type HostMetrics struct {
	// Память Go
	AllocBytes     uint64 `json:"alloc_bytes"`
	TotalAllocBytes uint64 `json:"total_alloc_bytes"`
	SysBytes       uint64 `json:"sys_bytes"`
	HeapAllocBytes uint64 `json:"heap_alloc_bytes"`
	HeapSysBytes   uint64 `json:"heap_sys_bytes"`
	NumGC          uint32 `json:"num_gc"`

	// Среда выполнения
	NumGoroutine int    `json:"num_goroutine"`
	NumCPU       int    `json:"num_cpu"`
	GoVersion    string `json:"go_version"`
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	Uptime       int64  `json:"uptime_seconds"`

	// Дисковое пространство библиотеки
	DiskTotalBytes uint64  `json:"disk_total_bytes"`
	DiskFreeBytes  uint64  `json:"disk_free_bytes"`
	DiskUsedBytes  uint64  `json:"disk_used_bytes"`
	DiskUsedPct    float64 `json:"disk_used_percent"`
}

// CollectHostMetrics собирает актуальные метрики хоста и диска библиотеки.
func CollectHostMetrics(libraryDir string) HostMetrics {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	totalDisk, freeDisk, _ := GetDiskUsage(libraryDir)
	usedDisk := uint64(0)
	usedPct := float64(0)
	if totalDisk > freeDisk {
		usedDisk = totalDisk - freeDisk
		if totalDisk > 0 {
			usedPct = float64(usedDisk) / float64(totalDisk) * 100.0
		}
	}

	return HostMetrics{
		AllocBytes:      mem.Alloc,
		TotalAllocBytes: mem.TotalAlloc,
		SysBytes:        mem.Sys,
		HeapAllocBytes:  mem.HeapAlloc,
		HeapSysBytes:    mem.HeapSys,
		NumGC:           mem.NumGC,

		NumGoroutine: runtime.NumGoroutine(),
		NumCPU:       runtime.NumCPU(),
		GoVersion:    runtime.Version(),
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		Uptime:       int64(time.Since(appStartTime).Seconds()),

		DiskTotalBytes: totalDisk,
		DiskFreeBytes:  freeDisk,
		DiskUsedBytes:  usedDisk,
		DiskUsedPct:    usedPct,
	}
}
