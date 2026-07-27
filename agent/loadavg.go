package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// loadAvg1 reads the 1-minute load average from /proc/loadavg (format:
// "<1m> <5m> <15m> <running>/<total> <last-pid>").
func loadAvg1() (float64, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, err
	}

	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0, fmt.Errorf("unexpected /proc/loadavg format: %q", string(data))
	}

	return strconv.ParseFloat(fields[0], 64)
}
