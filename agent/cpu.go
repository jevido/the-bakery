package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// cpuSampleInterval is the delta window used to compute CPU%: a point-in-time
// read of /proc/stat is a cumulative counter, not a percentage, so two
// samples are needed.
const cpuSampleInterval = 500 * time.Millisecond

type cpuSample struct {
	idle  uint64
	total uint64
}

func readCPUSample() (cpuSample, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return cpuSample{}, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return cpuSample{}, fmt.Errorf("read /proc/stat: empty")
	}

	fields := strings.Fields(scanner.Text())
	if len(fields) < 5 || fields[0] != "cpu" {
		return cpuSample{}, fmt.Errorf("unexpected /proc/stat format: %q", scanner.Text())
	}

	values := make([]uint64, 0, len(fields)-1)
	var total uint64
	for _, raw := range fields[1:] {
		v, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return cpuSample{}, fmt.Errorf("parse /proc/stat field %q: %w", raw, err)
		}
		values = append(values, v)
		total += v
	}

	// Fields are: user nice system idle iowait irq softirq steal guest guest_nice
	idle := values[3]
	if len(values) > 4 {
		idle += values[4] // iowait counts as idle
	}

	return cpuSample{idle: idle, total: total}, nil
}

// cpuPercent samples /proc/stat twice, cpuSampleInterval apart, and returns
// the percentage of non-idle CPU time across all cores in that window.
func cpuPercent(interval time.Duration) (float64, error) {
	first, err := readCPUSample()
	if err != nil {
		return 0, err
	}

	time.Sleep(interval)

	second, err := readCPUSample()
	if err != nil {
		return 0, err
	}

	totalDelta := second.total - first.total
	if totalDelta == 0 {
		return 0, nil
	}
	idleDelta := second.idle - first.idle

	return clampPct(float64(totalDelta-idleDelta) / float64(totalDelta) * 100), nil
}
