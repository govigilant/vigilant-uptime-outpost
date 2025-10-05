package httpserver

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"vigilant-uptime-outpost/internal/checks"
	"vigilant-uptime-outpost/internal/config"
	"vigilant-uptime-outpost/internal/registrar"
)

type Server struct {
	cfg      *config.Config
	checker  *checks.Checker
	registrar *registrar.Registrar
	server   *http.Server
}

func New(cfg *config.Config, c *checks.Checker, r *registrar.Registrar) *Server {
	s := &Server{cfg: cfg, checker: c, registrar: r}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.localhostOnly(s.health))
	mux.HandleFunc("/run-check", s.requireAuth(s.runCheck))
	s.server = &http.Server{
		Addr:    ":" + strconv.Itoa(cfg.Port),
		Handler: mux,
	}
	return s
}

func (s *Server) Start() error {
	log.Printf("listening on :%d", s.cfg.Port)
	return s.server.ListenAndServe()
}

func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.server.Shutdown(ctx)
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
	var job checks.Job
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	go func() {
		res := s.checker.Run(context.Background(), job)
		log.Printf("check result: %+v", res)
	}()
	w.WriteHeader(http.StatusAccepted)
}
