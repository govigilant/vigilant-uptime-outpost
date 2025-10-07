package checks

import (
	"context"
	"net"
	"time"

	"vigilant-uptime-outpost/internal/registrar"
)

func runTCP(ctx context.Context, reg registrar.Registration, job Job) Result {
	start := time.Now()
	
	timeout := 30 * time.Second
	if job.Timeout > 0 {
		timeout = time.Duration(job.Timeout) * time.Second
	}
	
	dialer := &net.Dialer{
		Timeout: timeout,
	}
	
	conn, err := dialer.DialContext(ctx, "tcp", job.Target)
	dur := time.Since(start).Seconds() * 1000
	
	if err != nil {
		return fail(job, reg, err)
	}
	defer conn.Close()
	
	return Result{
		Outpost:   reg,
		Type:      job.Type,
		Target:    job.Target,
		Up:        true,
		LatencyMS: dur,
		Timestamp: time.Now().UTC(),
	}
}
