package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// agentVersion is reported to the control plane on every check-in and
// compared against agentRelease rows to decide whether to self-update. A
// var (not const) so real builds can override it via
// `-ldflags "-X main.agentVersion=1.2.3"`.
var agentVersion = "0.1.0-dev"

// checkinPayload mirrors src/lib/server/agent/protocol.ts. Field names are
// wire-format contract shared with already-deployed agent binaries; keep
// them in sync with that schema, not the other way around.
type checkinPayload struct {
	CPUPct         float64         `json:"cpuPct"`
	MemPct         float64         `json:"memPct"`
	DiskPct        float64         `json:"diskPct"`
	PodmanVersion  string          `json:"podmanVersion"`
	ContainerCount int             `json:"containerCount"`
	AgentVersion   string          `json:"agentVersion"`
	Volumes        []volumeReport  `json:"volumes"`
	Containers     []containerStat `json:"containers"`
	Logs           []logLineReport `json:"logs"`
}

// pendingCommand mirrors PendingCommand in src/lib/server/agent/protocol.ts.
// `Payload` is left raw since its shape depends on `Type` — see commands.go.
type pendingCommand struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type checkinResponse struct {
	PendingCommands []pendingCommand `json:"pendingCommands"`
}

func checkin(ctx context.Context, httpClient *http.Client, cfg config, payload checkinPayload) (*checkinResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode payload: %w", err)
	}

	endpoint := strings.TrimSuffix(cfg.controlPlaneURL, "/") + "/api/v1/agent/checkin"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("check-in rejected: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed checkinResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &parsed, nil
}

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}
