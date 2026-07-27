package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// uptimeSeconds reads system uptime from /proc/uptime (format: "<uptime>
// <idle-time>", both in seconds).
func uptimeSeconds() (float64, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}

	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0, fmt.Errorf("unexpected /proc/uptime format: %q", string(data))
	}

	return strconv.ParseFloat(fields[0], 64)
}
