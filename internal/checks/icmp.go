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
	"sync"
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

	type pingResult struct {
		latency float64
		err     error
	}

	results := make(chan pingResult, pingAttempts)
	var wg sync.WaitGroup
	wg.Add(pingAttempts)

	for i := 0; i < pingAttempts; i++ {
		attempt := i + 1
		go func(attempt int) {
			defer wg.Done()
			latency, err := pingOnce(childCtx, target, timeoutSeconds, attempt)
			results <- pingResult{latency: latency, err: err}
		}(attempt)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var (
		sumLatencies float64
		successCount int
		firstErr     error
	)

	for res := range results {
		if res.err != nil {
			if firstErr == nil {
				firstErr = res.err
				cancel()
			}
			continue
		}
		successCount++
		sumLatencies += res.latency
	}

	if firstErr != nil {
		log.Printf("icmp check failed for %s: %v", target, firstErr)
		return fail(job, reg, firstErr)
	}

	if successCount == 0 {
		err := fmt.Errorf("all ping attempts failed")
		log.Printf("icmp check failed for %s: %v", target, err)
		return fail(job, reg, err)
	}

	avgLatency := sumLatencies / float64(successCount)

	return Result{
		Outpost:   reg,
		Type:      job.Type,
		Target:    target,
		Up:        true,
		LatencyMS: avgLatency,
		Timestamp: time.Now().UTC(),
	}
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
			return 0, err
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
