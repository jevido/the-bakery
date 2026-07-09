package main

import (
	"context"
	"log"
)

const diskPath = "/"

// collectPayload gathers real host and Podman metrics for one check-in
// cycle. Any single metric that can't be read is logged and reported as a
// zero/placeholder value rather than aborting the whole check-in — a host
// mid-setup (e.g. Podman not installed yet) should still report what it
// can.
func collectPayload(ctx context.Context) checkinPayload {
	cpu, err := cpuPercent(cpuSampleInterval)
	if err != nil {
		log.Printf("cpu metrics unavailable: %v", err)
	}

	mem, err := memPercent()
	if err != nil {
		log.Printf("memory metrics unavailable: %v", err)
	}

	disk, err := diskPercent(diskPath)
	if err != nil {
		log.Printf("disk metrics unavailable: %v", err)
	}

	podmanVersion := "unknown"
	containerCount := 0
	if pm, err := collectPodmanMetrics(ctx); err != nil {
		log.Printf("podman metrics unavailable: %v", err)
	} else {
		podmanVersion = pm.version
		containerCount = pm.containerCount
	}

	return checkinPayload{
		CPUPct:         cpu,
		MemPct:         mem,
		DiskPct:        disk,
		PodmanVersion:  podmanVersion,
		ContainerCount: containerCount,
		AgentVersion:   agentVersion,
	}
}
