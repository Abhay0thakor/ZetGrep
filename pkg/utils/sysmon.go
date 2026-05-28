package utils

import (
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

type SystemStats struct {
	CPUUsage    float64
	MemoryUsage float64
	RAMPercent  float64
}

func GetSystemStats() (SystemStats, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return SystemStats{}, err
	}

	c, err := cpu.Percent(time.Second, false)
	if err != nil {
		return SystemStats{}, err
	}

	cpuPct := 0.0
	if len(c) > 0 {
		cpuPct = c[0]
	}

	return SystemStats{
		CPUUsage:    cpuPct,
		MemoryUsage: float64(v.Used),
		RAMPercent:  v.UsedPercent,
	}, nil
}

func AutoScaleMonitor(maxRAM float64, onThrottle, onResume func()) {
	isThrottled := false
	for {
		stats, err := GetSystemStats()
		if err == nil {
			// If RAM usage > threshold, throttle
			if stats.RAMPercent >= maxRAM && !isThrottled {
				isThrottled = true
				if onThrottle != nil {
					onThrottle()
				}
			} else if stats.RAMPercent < (maxRAM-10.0) && isThrottled {
				isThrottled = false
				if onResume != nil {
					onResume()
				}
			}
		}
		time.Sleep(5 * time.Second)
	}
}
