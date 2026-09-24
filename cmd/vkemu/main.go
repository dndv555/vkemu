package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"vkemu/internal/api"
	"vkemu/internal/config"
	"vkemu/internal/ratelimit"
	"vkemu/internal/store"
	"vkemu/internal/web"
)

func main() {
	cfg := config.Load()

	db, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer db.Close()

	if err := db.Bootstrap(); err != nil {
		log.Fatalf("bootstrap: %v", err)
	}

	apiServer := api.NewServer(cfg, db)
	webServer := web.New(db, cfg.UploadDir)

	webLimit := ratelimit.New(ratelimit.Options{
		Rate:       cfg.RateLimit,
		Burst:      cfg.RateBurst,
		MaxConns:   cfg.MaxConns,
		TrustProxy: cfg.TrustProxy,
	})

	log.Printf("api listening on %s", cfg.Addr)
	log.Printf("api endpoint:   %s/api.php", cfg.BaseURL)
	log.Printf("long poll host: %s", cfg.LongPollHost)
	log.Printf("web version:    http://localhost%s", cfg.WebAddr)
	log.Printf("sqlite database: %s", cfg.DBPath)
	log.Printf("anti-ddos:      %.0f req/s per ip, burst %.0f, max conns %d", cfg.RateLimit, cfg.RateBurst, cfg.MaxConns)

	apiHTTP := &http.Server{Addr: cfg.Addr, Handler: apiServer.Handler()}
	webHTTP := &http.Server{Addr: cfg.WebAddr, Handler: webLimit.Middleware(webServer.Handler())}

	go func() {
		if err := apiHTTP.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("api serve: %v", err)
		}
	}()
	go func() {
		if err := webHTTP.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("web serve: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("shutting down")
	apiHTTP.Close()
	webHTTP.Close()
}
