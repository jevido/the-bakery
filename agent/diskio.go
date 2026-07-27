package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

// diskIOSampleInterval mirrors cpuSampleInterval: /proc/diskstats reports
// cumulative sector counters, not a rate, so two samples are needed.
const diskIOSampleInterval = 500 * time.Millisecond

// sectorSizeBytes is fixed by the kernel's /proc/diskstats convention --
// always 512, regardless of a device's real physical sector size.
const sectorSizeBytes = 512

type diskIOSample struct {
	readSectors  uint64
	writeSectors uint64
}

// isWholeDisk reports whether name is a top-level block device (e.g. "sda",
// "nvme0n1") rather than a partition (e.g. "sda1") -- /sys/block only
// contains whole-disk entries, so checking there is a simple, reliable way
// to avoid double-counting a disk's I/O once for the whole device and again
// for each of its partitions, without device-name pattern matching.
func isWholeDisk(name string) bool {
	info, err := os.Stat("/sys/block/" + name)
	return err == nil && info.IsDir()
}

// isVirtualDisk excludes device types that don't represent real hardware
// I/O (loopback images, ramdisks, device-mapper overlays) -- a dashboard
// showing "disk I/O throughput" means physical disk activity, not software
// layers on top of it.
func isVirtualDisk(name string) bool {
	return strings.HasPrefix(name, "loop") ||
		strings.HasPrefix(name, "ram") ||
		strings.HasPrefix(name, "dm-") ||
		strings.HasPrefix(name, "zram")
}

func readDiskIOSample() (diskIOSample, error) {
	f, err := os.Open("/proc/diskstats")
	if err != nil {
		return diskIOSample{}, err
	}
	defer f.Close()

	var sample diskIOSample
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		// Fields: major minor name reads_completed reads_merged
		// sectors_read time_reading writes_completed writes_merged
		// sectors_written ...
		fields := strings.Fields(scanner.Text())
		if len(fields) < 10 {
			continue
		}
		name := fields[2]
		if !isWholeDisk(name) || isVirtualDisk(name) {
			continue
		}
		readSectors, err := strconv.ParseUint(fields[5], 10, 64)
		if err != nil {
			continue
		}
		writeSectors, err := strconv.ParseUint(fields[9], 10, 64)
		if err != nil {
			continue
		}
		sample.readSectors += readSectors
		sample.writeSectors += writeSectors
	}
	if err := scanner.Err(); err != nil {
		return diskIOSample{}, err
	}

	return sample, nil
}

// diskIOBytesPerSec samples /proc/diskstats twice, diskIOSampleInterval
// apart, and returns read/write throughput in bytes/sec summed across every
// physical disk on the host.
func diskIOBytesPerSec(interval time.Duration) (readBps, writeBps float64, err error) {
	first, err := readDiskIOSample()
	if err != nil {
		return 0, 0, err
	}

	time.Sleep(interval)

	second, err := readDiskIOSample()
	if err != nil {
		return 0, 0, err
	}

	seconds := interval.Seconds()
	readBps = float64(second.readSectors-first.readSectors) * sectorSizeBytes / seconds
	writeBps = float64(second.writeSectors-first.writeSectors) * sectorSizeBytes / seconds
	return readBps, writeBps, nil
}
