//go:build linux

package main

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

var (
	linuxCPUMu     sync.Mutex
	linuxPrevTotal uint64
	linuxPrevIdle  uint64
)

func collectMetrics(dataDir string) *domain.MachineMetrics {
	metrics := &domain.MachineMetrics{
		CPUPercent:  linuxCPUPercent(),
		CollectedAt: time.Now().UTC(),
	}
	if total, available, err := linuxMemory(); err == nil {
		metrics.MemoryTotal = total
		metrics.MemoryUsed = total - available
	}
	path := dataDir
	if path == "" {
		path = "."
	}
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err == nil {
		metrics.DiskTotal = stat.Blocks * uint64(stat.Bsize)
		metrics.DiskUsed = (stat.Blocks - stat.Bavail) * uint64(stat.Bsize)
	}
	return metrics
}

func linuxCPUPercent() float64 {
	raw, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	lines := strings.SplitN(string(raw), "\n", 2)
	fields := strings.Fields(lines[0])
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0
	}
	var total uint64
	for _, field := range fields[1:] {
		value, _ := strconv.ParseUint(field, 10, 64)
		total += value
	}
	idle, _ := strconv.ParseUint(fields[4], 10, 64)

	linuxCPUMu.Lock()
	defer linuxCPUMu.Unlock()
	if linuxPrevTotal == 0 {
		linuxPrevTotal, linuxPrevIdle = total, idle
		return 0
	}
	deltaTotal := total - linuxPrevTotal
	deltaIdle := idle - linuxPrevIdle
	linuxPrevTotal, linuxPrevIdle = total, idle
	if deltaTotal == 0 {
		return 0
	}
	return float64(deltaTotal-deltaIdle) / float64(deltaTotal) * 100
}

func linuxMemory() (uint64, uint64, error) {
	raw, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	values := map[string]uint64{}
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value, _ := strconv.ParseUint(fields[1], 10, 64)
		values[strings.TrimSuffix(fields[0], ":")] = value * 1024
	}
	total := values["MemTotal"]
	available, ok := values["MemAvailable"]
	if !ok {
		available = values["MemFree"] + values["Buffers"] + values["Cached"]
	}
	return total, available, nil
}
