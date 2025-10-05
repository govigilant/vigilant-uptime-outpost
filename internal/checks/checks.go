package checks

import (
	"context"
	"time"

	"vigilant-uptime-outpost/internal/registrar"
)

type Job struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Target      string            `json:"target"`
	Method      string            `json:"method,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        string            `json:"body,omitempty"`
	TimeoutSec  int               `json:"timeout_seconds,omitempty"`
	CallbackURL string            `json:"callback_url,omitempty"`
}

type Result struct {
	ID         string              `json:"id"`
	Outpost    registrar.Registration `json:"outpost"`
	Type       string              `json:"type"`
	Target     string              `json:"target"`
	OK         bool                `json:"ok"`
	LatencyMS  float64             `json:"latency_ms"`
	StatusCode int                 `json:"status_code,omitempty"`
	Error      string              `json:"error,omitempty"`
	Timestamp  time.Time           `json:"timestamp"`
}

type Checker struct {
	reg registrar.Registration
}

func New(reg *registrar.Registrar) *Checker {
	return &Checker{reg: reg.Info()}
}

func (c *Checker) Run(ctx context.Context, job Job) Result {
	switch job.Type {
	case "http":
		return runHTTP(ctx, c.reg, job)
	case "tcp":
		return runTCP(ctx, c.reg, job)
	case "icmp":
		return runICMP(ctx, c.reg, job)
	default:
		return Result{
			ID: job.ID, Outpost: c.reg, Type: job.Type, Target: job.Target,
			Error: "unknown check type",
		}
	}
}
