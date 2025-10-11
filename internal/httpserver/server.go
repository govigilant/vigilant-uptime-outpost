package httpserver

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"vigilant-uptime-outpost/internal/checks"
	"vigilant-uptime-outpost/internal/config"
	"vigilant-uptime-outpost/internal/registrar"
)

type Server struct {
	cfg           *config.Config
	checker       *checks.Checker
	registrar     *registrar.Registrar
	server        *http.Server
	lastRequest   time.Time
	lastRequestMu sync.RWMutex
	shutdownChan  chan struct{}
}

func New(cfg *config.Config, c *checks.Checker, r *registrar.Registrar) *Server {
	s := &Server{
		cfg:          cfg,
		checker:      c,
		registrar:    r,
		lastRequest:  time.Now(),
		shutdownChan: make(chan struct{}),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.localhostOnly(s.health))
	mux.HandleFunc("/run-check", s.requireAuth(s.trackActivity(s.runCheck)))
	s.server = &http.Server{
		Addr:    ":" + strconv.Itoa(cfg.Port),
		Handler: mux,
	}
	return s
}

func (s *Server) Start() error {
	certData := s.registrar.GetCertificates()
	
	// Start inactivity monitor
	go s.monitorInactivity()
	
	// If we have certificates, start HTTPS server
	if certData != nil && certData.Certificate != "" && certData.PrivateKey != "" {
		log.Printf("starting HTTPS server on :%d", s.cfg.Port)
		
		// Create certificate from PEM data
		cert, err := tls.X509KeyPair([]byte(certData.Certificate), []byte(certData.PrivateKey))
		if err != nil {
			log.Printf("failed to load certificate: %v", err)
			return err
		}
		
		// Configure TLS
		s.server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		
		return s.server.ListenAndServeTLS("", "")
	}
	
	// Fall back to HTTP if no certificates
	log.Printf("starting HTTP server on :%d", s.cfg.Port)
	return s.server.ListenAndServe()
}

func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.server.Shutdown(ctx)
}

func (s *Server) GetShutdownChan() <-chan struct{} {
	return s.shutdownChan
}

func (s *Server) trackActivity(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.lastRequestMu.Lock()
		s.lastRequest = time.Now()
		s.lastRequestMu.Unlock()
		next(w, r)
	}
}

func (s *Server) monitorInactivity() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	inactivityTimeout := 1 * time.Hour
	
	for range ticker.C {
		s.lastRequestMu.RLock()
		lastReq := s.lastRequest
		s.lastRequestMu.RUnlock()
		
		if time.Since(lastReq) > inactivityTimeout {
			log.Printf("no requests received for %v, initiating shutdown for restart", inactivityTimeout)
			close(s.shutdownChan)
			return
		}
	}
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.OutpostSecret == "" {
			next(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if parts[1] != s.cfg.OutpostSecret {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func (s *Server) localhostOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) runCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}

	var body json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Try to parse as array first (batch request)
	var jobs []checks.Job
	if err := json.Unmarshal(body, &jobs); err == nil && len(jobs) > 0 {
		// Handle batch request
		results := s.runBatchChecks(r.Context(), jobs)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
		return
	}

	// Parse as single job
	var job checks.Job
	if err := json.Unmarshal(body, &job); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Run single check synchronously
	result := s.checker.Run(r.Context(), job)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) runBatchChecks(ctx context.Context, jobs []checks.Job) []checks.Result {
	results := make([]checks.Result, len(jobs))
	
	// Use a channel to collect results from concurrent checks
	type indexedResult struct {
		index  int
		result checks.Result
	}
	resultChan := make(chan indexedResult, len(jobs))

	// Run all checks concurrently
	for i, job := range jobs {
		go func(idx int, j checks.Job) {
			resultChan <- indexedResult{
				index:  idx,
				result: s.checker.Run(ctx, j),
			}
		}(i, job)
	}

	// Collect results
	for range jobs {
		ir := <-resultChan
		results[ir.index] = ir.result
	}

	return results
}
