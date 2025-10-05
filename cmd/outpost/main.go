package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"github.com/joho/godotenv"

	"vigilant-uptime-outpost/internal/checks"
	"vigilant-uptime-outpost/internal/config"
	"vigilant-uptime-outpost/internal/httpserver"
	"vigilant-uptime-outpost/internal/registrar"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found")
	}
	cfg := config.Load()

	reg := registrar.New(cfg)
	checker := checks.New(reg)
	server := httpserver.New(cfg, checker, reg)

	ctx, cancel := context.WithCancel(context.Background())
	if err := reg.Register(ctx); err != nil {
		log.Printf("registration failed: %v", err)
		os.Exit(1)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			log.Printf("server error: %v", err)
			sig <- syscall.SIGTERM
		}
	}()

	<-sig
	cancel()
	
	log.Println("shutting down outpost...")
	
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	
	if err := reg.Unregister(shutdownCtx); err != nil {
		log.Printf("unregister error: %v", err)
	}
	
	server.Stop()
	log.Println("outpost stopped")
}
