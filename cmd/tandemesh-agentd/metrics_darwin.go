//go:build darwin

package main

import (
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

var (
	darwinCPUMu     sync.Mutex
	darwinPrevTotal uint64
	darwinPrevIdle  uint64
)

func collectMetrics(dataDir string) *domain.MachineMetrics {
	metrics := &domain.MachineMetrics{
		CPUPercent:  darwinCPUPercent(),
		CollectedAt: time.Now().UTC(),
	}
	if total, err := sysctlUint64("hw.memsize"); err == nil {
		metrics.MemoryTotal = total
		if available, err := darwinAvailableMemory(); err == nil {
			metrics.MemoryUsed = total - available
		}
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

func darwinCPUPercent() float64 {
	raw, err := exec.Command("sysctl", "-n", "kern.cp_time").Output()
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(raw))
	if len(fields) < 4 {
		return 0
	}
	var total uint64
	values := make([]uint64, 0, len(fields))
	for _, field := range fields {
		value, _ := strconv.ParseUint(field, 10, 64)
		values = append(values, value)
		total += value
	}
	idle := values[3]

	darwinCPUMu.Lock()
	defer darwinCPUMu.Unlock()
	if darwinPrevTotal == 0 {
		darwinPrevTotal, darwinPrevIdle = total, idle
		return 0
	}
	deltaTotal := total - darwinPrevTotal
	deltaIdle := idle - darwinPrevIdle
	darwinPrevTotal, darwinPrevIdle = total, idle
	if deltaTotal == 0 {
		return 0
	}
	return float64(deltaTotal-deltaIdle) / float64(deltaTotal) * 100
}

func sysctlUint64(name string) (uint64, error) {
	raw, err := exec.Command("sysctl", "-n", name).Output()
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(strings.TrimSpace(string(raw)), 10, 64)
}

func darwinAvailableMemory() (uint64, error) {
	raw, err := exec.Command("vm_stat").Output()
	if err != nil {
		return 0, err
	}
	pageSize := uint64(4096)
	available := uint64(0)
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.Contains(line, "page size of") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "of" && i+1 < len(fields) {
					if size, err := strconv.ParseUint(fields[i+1], 10, 64); err == nil {
						pageSize = size
					}
				}
			}
			continue
		}
		for _, key := range []string{"Pages free:", "Pages inactive:", "Pages speculative:"} {
			if strings.HasPrefix(strings.TrimSpace(line), key) {
				fields := strings.Fields(line)
				if len(fields) >= 3 {
					if value, err := strconv.ParseUint(strings.TrimSuffix(fields[len(fields)-1], "."), 10, 64); err == nil {
						available += value
					}
				}
			}
		}
	}
	return available * pageSize, nil
}
