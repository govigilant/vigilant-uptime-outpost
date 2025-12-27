package checks

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"vigilant-uptime-outpost/internal/registrar"
)

func runICMP(ctx context.Context, reg registrar.Registration, job Job) Result {
	start := time.Now()

	target, err := sanitizePingTarget(job.Target)
	if err != nil {
		return fail(job, reg, err)
	}

	timeout := jobTimeoutDuration(job)
	timeoutSeconds := int(timeout.Seconds())
	if timeoutSeconds < 1 {
		timeoutSeconds = 1
	}

	cmd := exec.CommandContext(ctx, "ping", "-c", "1", "-w", strconv.Itoa(timeoutSeconds), target)
	output, err := cmd.CombinedOutput()
	dur := time.Since(start).Seconds() * 1000

	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed != "" {
			err = fmt.Errorf("ping failed: %w: %s", err, trimmed)
		}
		return fail(job, reg, err)
	}

	return Result{
		Outpost:   reg,
		Type:      job.Type,
		Target:    job.Target,
		Up:        true,
		LatencyMS: dur,
		Timestamp: time.Now().UTC(),
	}
}

func sanitizePingTarget(raw string) (string, error) {
	target := strings.TrimSpace(raw)
	if target == "" {
		return "", fmt.Errorf("target is required")
	}

	if net.ParseIP(target) != nil {
		return target, nil
	}

	target = strings.TrimSuffix(target, ".")
	if isValidHostname(target) {
		return target, nil
	}

	return "", fmt.Errorf("invalid target: %q", raw)
}

func isValidHostname(host string) bool {
	if host == "" || len(host) > 253 {
		return false
	}

	labels := strings.Split(host, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
				continue
			}
			return false
		}
	}

	return true
}
