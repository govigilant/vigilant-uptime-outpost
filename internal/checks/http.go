package checks

import (
	"context"
	"net/http"
	"strings"
	"time"

	"vigilant-uptime-outpost/internal/registrar"
)

func runHTTP(ctx context.Context, reg registrar.Registration, job Job) Result {
	start := time.Now()
	method := "GET"
	if job.Method != "" {
		method = job.Method
	}

	req, err := http.NewRequestWithContext(ctx, method, job.Target, strings.NewReader(job.Body))
	if err != nil {
		return fail(job, reg, err)
	}
	for k, v := range job.Headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	dur := time.Since(start).Seconds() * 1000
	if err != nil {
		return fail(job, reg, err)
	}
	defer resp.Body.Close()

	ok := resp.StatusCode >= 200 && resp.StatusCode < 400
	return Result{
		Outpost: reg, Type: job.Type, Target: job.Target,
		OK: ok, LatencyMS: dur, StatusCode: resp.StatusCode,
		Timestamp: time.Now().UTC(),
	}
}

func fail(job Job, reg registrar.Registration, err error) Result {
	return Result{
		Outpost: reg, Type: job.Type, Target: job.Target,
		Error: err.Error(), Timestamp: time.Now().UTC(),
	}
}
