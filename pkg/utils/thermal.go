package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

type ThermalStatus struct {
	Temperature float64
	IsCritical  bool
}

func GetCPUTemperature() (float64, error) {
	// Linux specific thermal zone check
	for i := 0; i < 10; i++ {
		path := fmt.Sprintf("/sys/class/thermal/thermal_zone%d/temp", i)
		if _, err := os.Stat(path); err == nil {
			data, err := ioutil.ReadFile(path)
			if err != nil {
				continue
			}
			tempStr := strings.TrimSpace(string(data))
			tempInt, err := strconv.ParseFloat(tempStr, 64)
			if err != nil {
				continue
			}
			// Most systems return temp in milli-Celsius
			return tempInt / 1000.0, nil
		}
	}
	return 0, fmt.Errorf("could not find thermal zone")
}

func MonitorThermal(criticalThreshold, safetyThreshold float64, onCritical, onSafe func()) {
	isPaused := false
	for {
		temp, err := GetCPUTemperature()
		if err == nil {
			if temp >= criticalThreshold && !isPaused {
				isPaused = true
				if onCritical != nil {
					onCritical()
				}
			} else if temp <= safetyThreshold && isPaused {
				isPaused = false
				if onSafe != nil {
					onSafe()
				}
			}
		}
		time.Sleep(10 * time.Second)
	}
}
