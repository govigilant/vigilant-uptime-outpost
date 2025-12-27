package checks

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"vigilant-uptime-outpost/internal/registrar"
)

func runICMP(ctx context.Context, reg registrar.Registration, job Job) Result {
	start := time.Now()

	timeout := jobTimeoutDuration(job)
	timeoutSeconds := int(timeout.Seconds())
	if timeoutSeconds < 1 {
		timeoutSeconds = 1
	}

	cmd := exec.CommandContext(ctx, "ping", "-c", "1", "-w", strconv.Itoa(timeoutSeconds), job.Target)
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
