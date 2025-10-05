package checks

import (
	"context"
	"fmt"
	"time"

	"vigilant-uptime-outpost/internal/registrar"
)

func runICMP(ctx context.Context, reg registrar.Registration, job Job) Result {
	// ICMP requires raw sockets which need elevated privileges
	// For now, return an error indicating this is not implemented
	return Result{
		ID:        job.ID,
		Outpost:   reg,
		Type:      job.Type,
		Target:    job.Target,
		OK:        false,
		Error:     fmt.Sprintf("ICMP checks not yet implemented"),
		Timestamp: time.Now().UTC(),
	}
}
