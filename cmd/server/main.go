package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"otklik/internal/config"
	"otklik/internal/db"
	"otklik/internal/httpapi"
	"otklik/internal/store"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := os.MkdirAll(cfg.AttachmentsDir, 0o755); err != nil {
		log.Fatalf("attachments dir: %v", err)
	}

	d, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer d.Close()

	if err := db.Migrate(ctx, d); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if err := db.Seed(ctx, d, cfg.SeedDefaultPwd); err != nil {
		log.Fatalf("seed: %v", err)
	}

	st := store.New(d)

	publicSrv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           httpapi.New(cfg, st),
		ReadHeaderTimeout: 10 * time.Second,
	}
	staffSrv := &http.Server{
		Addr:              cfg.StaffListenAddr,
		Handler:           httpapi.NewStaff(cfg, st),
		ReadHeaderTimeout: 10 * time.Second,
	}

	run := func(name string, srv *http.Server) {
		go func() {
			log.Printf("otklik: %s listening on %s", name, srv.Addr)
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Fatalf("listen: %v", err)
			}
		}()
	}
	run("public (заявители)", publicSrv)
	run("staff (сотрудники)", staffSrv)

	<-ctx.Done()
	log.Println("otklik: shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := publicSrv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	if err := staffSrv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
