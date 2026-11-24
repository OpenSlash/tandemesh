//go:build windows

package main

import (
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

var (
	kernel32                 = syscall.NewLazyDLL("kernel32.dll")
	procGetSystemTimes       = kernel32.NewProc("GetSystemTimes")
	procGlobalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")
	cpuMu                    sync.Mutex
	cpuPrevIdle              uint64
	cpuPrevKernel            uint64
	cpuPrevUser              uint64
)

type filetime struct {
	low  uint32
	high uint32
}

func (f filetime) uint64() uint64 {
	return uint64(f.high)<<32 | uint64(f.low)
}

type memoryStatusEx struct {
	length               uint32
	memoryLoad           uint32
	totalPhys            uint64
	availPhys            uint64
	totalPageFile        uint64
	availPageFile        uint64
	totalVirtual         uint64
	availVirtual         uint64
	availExtendedVirtual uint64
}

func collectMetrics(dataDir string) *domain.MachineMetrics {
	metrics := &domain.MachineMetrics{
		CPUPercent:  cpuPercent(),
		CollectedAt: time.Now().UTC(),
	}

	var mem memoryStatusEx
	mem.length = uint32(unsafe.Sizeof(mem))
	if ret, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&mem))); ret != 0 {
		metrics.MemoryTotal = mem.totalPhys
		metrics.MemoryUsed = mem.totalPhys - mem.availPhys
	}

	path := dataDir
	if path == "" {
		path = "."
	}
	if ptr, err := windows.UTF16PtrFromString(path); err == nil {
		var free, total, totalFree uint64
		if err := windows.GetDiskFreeSpaceEx(ptr, &free, &total, &totalFree); err == nil {
			metrics.DiskTotal = total
			metrics.DiskUsed = total - totalFree
		}
	}
	return metrics
}

func cpuPercent() float64 {
	var idle, kernel, user filetime
	ret, _, _ := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idle)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	)
	if ret == 0 {
		return 0
	}
	curIdle := idle.uint64()
	curKernel := kernel.uint64()
	curUser := user.uint64()

	cpuMu.Lock()
	defer cpuMu.Unlock()
	if cpuPrevKernel == 0 && cpuPrevUser == 0 {
		cpuPrevIdle, cpuPrevKernel, cpuPrevUser = curIdle, curKernel, curUser
		return 0
	}
	deltaIdle := curIdle - cpuPrevIdle
	deltaKernel := curKernel - cpuPrevKernel
	deltaUser := curUser - cpuPrevUser
	cpuPrevIdle, cpuPrevKernel, cpuPrevUser = curIdle, curKernel, curUser

	total := deltaKernel + deltaUser
	if total == 0 {
		return 0
	}
	busy := total - deltaIdle
	return float64(busy) / float64(total) * 100
}
