package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func memPercent() (float64, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var total, available float64
	haveAvailable := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "MemTotal:"):
			total = parseMeminfoKB(line)
		case strings.HasPrefix(line, "MemAvailable:"):
			available = parseMeminfoKB(line)
			haveAvailable = true
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	if total == 0 {
		return 0, fmt.Errorf("could not determine MemTotal from /proc/meminfo")
	}
	if !haveAvailable {
		return 0, fmt.Errorf("could not determine MemAvailable from /proc/meminfo")
	}

	return clampPct((total - available) / total * 100), nil
}

// swapPercent reports swap usage as a percentage of total swap configured.
// Returns 0 (not an error) when the host has no swap at all -- that's a
// legitimate, common configuration, not a failure to read the metric.
func swapPercent() (float64, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var total, free float64
	haveTotal, haveFree := false, false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "SwapTotal:"):
			total = parseMeminfoKB(line)
			haveTotal = true
		case strings.HasPrefix(line, "SwapFree:"):
			free = parseMeminfoKB(line)
			haveFree = true
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	if !haveTotal || !haveFree {
		return 0, fmt.Errorf("could not determine swap usage from /proc/meminfo")
	}
	if total == 0 {
		return 0, nil
	}

	return clampPct((total - free) / total * 100), nil
}

func parseMeminfoKB(line string) float64 {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}
	v, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return 0
	}
	return v
}
