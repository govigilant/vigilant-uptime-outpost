package checks

import (
	"context"
	"fmt"
	"log"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"vigilant-uptime-outpost/internal/registrar"
)

const pingAttempts = 3

var (
	pingRTTRegex     = regexp.MustCompile(`time[=:]\s*(\d+(?:\.\d+)?)\s*ms`)
	pingSummaryRegex = regexp.MustCompile(`(?m)(?:rtt|round-trip)\s+min/avg/max/(?:mdev|stddev)\s*=\s*(\d+(?:\.\d+)?)/(\d+(?:\.\d+)?)/(\d+(?:\.\d+)?)/(\d+(?:\.\d+)?)\s*ms`)
)

func runICMP(ctx context.Context, reg registrar.Registration, job Job) Result {
	target, err := sanitizePingTarget(job.Target)
	if err != nil {
		return fail(job, reg, err)
	}

	timeout := jobTimeoutDuration(job)
	timeoutSeconds := int(timeout.Seconds())
	if timeoutSeconds < 1 {
		timeoutSeconds = 1
	}

	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var lastErr error
	for attempt := 1; attempt <= pingAttempts; attempt++ {
		latency, err := pingOnce(childCtx, target, timeoutSeconds, attempt)
		if err == nil {
			return Result{
				Outpost:   reg,
				Type:      job.Type,
				Target:    target,
				Up:        true,
				LatencyMS: latency,
				Timestamp: time.Now().UTC(),
			}
		}
		lastErr = err
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("all ping attempts failed")
	} else {
		lastErr = fmt.Errorf("all ping attempts failed: %w", lastErr)
	}

	log.Printf("icmp check failed for %s: %v", target, lastErr)
	return fail(job, reg, lastErr)
}

func pingOnce(ctx context.Context, target string, timeoutSeconds int, attempt int) (float64, error) {
	cmd := exec.CommandContext(ctx, "ping", "-c", "1", "-w", strconv.Itoa(timeoutSeconds), target)
	output, err := cmd.CombinedOutput()

	trimmed := strings.TrimSpace(string(output))

	if err != nil {
		if trimmed != "" {
			err = fmt.Errorf("ping failed: %w: %s", err, trimmed)
		}
		log.Printf("icmp ping attempt %d error for %s: %v", attempt, target, err)
		return 0, err
	}

	latency, err := parseRTT(string(output))
	if err != nil {
		log.Printf("icmp ping attempt %d parse error for %s: %v", attempt, target, err)
		return 0, fmt.Errorf("failed to parse ping RTT: %w", err)
	}

	return latency, nil
}

func parseRTT(output string) (float64, error) {
	if summary := pingSummaryRegex.FindStringSubmatch(output); len(summary) == 5 {
		if avg, err := strconv.ParseFloat(summary[2], 64); err == nil {
			return avg, nil
		}
	}

	matches := pingRTTRegex.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return 0, fmt.Errorf("RTT not found in ping output")
	}

	var sum float64
	for _, match := range matches {
		val, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse RTT value %q: %w", match[1], err)
		}
		sum += val
	}

	return sum / float64(len(matches)), nil
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
