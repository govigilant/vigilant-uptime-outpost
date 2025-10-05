package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"github.com/joho/godotenv"

	"vigilant-uptime-outpost/internal/checks"
	"vigilant-uptime-outpost/internal/config"
	"vigilant-uptime-outpost/internal/httpserver"
	"vigilant-uptime-outpost/internal/registrar"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found or failed to load: %v", err)
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

	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	cancel()
	server.Stop()
	log.Println("outpost stopped")
}
