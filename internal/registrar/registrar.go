package registrar

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"vigilant-uptime-outpost/internal/config"
)

type Registration struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Hostname string `json:"hostname"`
}

type Registrar struct {
	cfg *config.Config
}

func New(cfg *config.Config) *Registrar {
	return &Registrar{cfg: cfg}
}

func (r *Registrar) Info() Registration {
	return Registration{
		IP:       r.cfg.IP,
		Port:     r.cfg.Port,
		Hostname: r.cfg.Hostname,
	}
}

func (r *Registrar) Register(ctx context.Context) error {
	if r.cfg.VigilantURL == "" {
		log.Println("VIGILANT_URL not set, skipping registration")
		return nil
	}
	log.Printf("registering with Vigilant at %s", r.cfg.VigilantURL)
	url := strings.TrimRight(r.cfg.VigilantURL, "/") + "/api/v1/outposts/register"
	body, _ := json.Marshal(Registration{
		IP: r.cfg.IP, Port: r.cfg.Port,
	})

	backoff := time.Second
	for {
		log.Printf("attempting to register with Vigilant at %s", url)
		req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if r.cfg.OutpostSecret != "" {
			req.Header.Set("Authorization", "Bearer "+r.cfg.OutpostSecret)
		}
		resp, err := http.DefaultClient.Do(req)
		if err == nil && resp.StatusCode < 300 {
			resp.Body.Close()
			log.Printf("registered with Vigilant at %s", url)
			return nil
		}
		log.Printf("error registering with Vigilant at %s: %v %s", url, err, resp.Status)
		if resp != nil {
			resp.Body.Close()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			if backoff < time.Minute {
				backoff *= 2
			}
		}
	}
}

func (r *Registrar) Unregister(ctx context.Context) error {
	if r.cfg.VigilantURL == "" {
		log.Println("VIGILANT_URL not set, skipping unregistration")
		return nil
	}
	log.Printf("unregistering from Vigilant at %s", r.cfg.VigilantURL)
	url := strings.TrimRight(r.cfg.VigilantURL, "/") + "/api/v1/outposts/unregister"
	body, _ := json.Marshal(Registration{
		IP: r.cfg.IP, Port: r.cfg.Port,
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if r.cfg.OutpostSecret != "" {
		req.Header.Set("Authorization", "Bearer "+r.cfg.OutpostSecret)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("error unregistering from Vigilant at %s: %v", url, err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("error unregistering from Vigilant at %s: status %s", url, resp.Status)
		return nil
	}
	log.Printf("unregistered from Vigilant at %s", url)
	return nil
}
