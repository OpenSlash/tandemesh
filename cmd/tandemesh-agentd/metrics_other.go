//go:build !windows && !linux && !darwin

package main

import (
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// Host metrics are currently sampled on Windows and Linux. Other platforms
// still report the timestamp and can be extended with platform collectors.
func collectMetrics(string) *domain.MachineMetrics {
	return &domain.MachineMetrics{CollectedAt: time.Now().UTC()}
}
