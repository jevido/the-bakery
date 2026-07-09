package main

import "syscall"

// diskPercent reports usage of the filesystem containing path (the root
// filesystem for the default "/").
func diskPercent(path string) (float64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	if total == 0 {
		return 0, nil
	}

	used := total - free
	return clampPct(float64(used) / float64(total) * 100), nil
}
