package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

// netIOSampleInterval mirrors cpuSampleInterval: /proc/net/dev reports
// cumulative byte counters, not a rate, so two samples are needed.
const netIOSampleInterval = 500 * time.Millisecond

type netIOSample struct {
	rxBytes uint64
	txBytes uint64
}

func readNetIOSample() (netIOSample, error) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return netIOSample{}, err
	}
	defer f.Close()

	var sample netIOSample
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum <= 2 {
			continue // two header lines
		}

		parts := strings.SplitN(scanner.Text(), ":", 2)
		if len(parts) != 2 {
			continue
		}
		iface := strings.TrimSpace(parts[0])
		if iface == "lo" {
			continue // loopback isn't real network I/O
		}

		// Fields after the colon: rx_bytes rx_packets rx_errs rx_drop
		// rx_fifo rx_frame rx_compressed rx_multicast tx_bytes ...
		fields := strings.Fields(parts[1])
		if len(fields) < 9 {
			continue
		}
		rx, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			continue
		}
		tx, err := strconv.ParseUint(fields[8], 10, 64)
		if err != nil {
			continue
		}
		sample.rxBytes += rx
		sample.txBytes += tx
	}
	if err := scanner.Err(); err != nil {
		return netIOSample{}, err
	}

	return sample, nil
}

// netIOBytesPerSec samples /proc/net/dev twice, netIOSampleInterval apart,
// and returns receive/transmit throughput in bytes/sec summed across every
// non-loopback interface on the host.
func netIOBytesPerSec(interval time.Duration) (rxBps, txBps float64, err error) {
	first, err := readNetIOSample()
	if err != nil {
		return 0, 0, err
	}

	time.Sleep(interval)

	second, err := readNetIOSample()
	if err != nil {
		return 0, 0, err
	}

	seconds := interval.Seconds()
	rxBps = float64(second.rxBytes-first.rxBytes) / seconds
	txBps = float64(second.txBytes-first.txBytes) / seconds
	return rxBps, txBps, nil
}
