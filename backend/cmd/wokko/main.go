// Command wokko runs the Wokko server. It also has a restore command for backups.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/serraniel/wokko/backend/internal/api"
	"github.com/serraniel/wokko/backend/internal/app"
	"github.com/serraniel/wokko/backend/internal/backup"
	"github.com/serraniel/wokko/backend/internal/config"
	"github.com/serraniel/wokko/backend/internal/store"
)

// version is set at build time from the git tag.
var version = "dev"

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg, err := config.Load(version)
	if err == nil {
		switch {
		case len(os.Args) > 1 && os.Args[1] == "restore":
			err = restore(cfg, os.Args[2:])
		case len(os.Args) > 1 && os.Args[1] == "healthcheck":
			err = healthcheck(cfg)
		case len(os.Args) > 1 && os.Args[1] == "version":
			fmt.Println(version)
		case len(os.Args) > 1:
			err = fmt.Errorf("unknown command %q. Commands: restore <backup file>, healthcheck, version", os.Args[1])
		default:
			err = serve(cfg, log)
		}
	}
	if err != nil {
		log.Error("Wokko stopped", "reason", err)
		os.Exit(1)
	}
}

func restore(cfg config.Config, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: wokko restore <backup file>")
	}
	kept, err := backup.Restore(args[0], cfg.DataDir, time.Now())
	if err != nil {
		return err
	}
	fmt.Printf("Restored %s.\nThe previous data was kept in %s.\n", args[0], kept)
	return nil
}

func healthcheck(cfg config.Config) error {
	client := http.Client{Timeout: 3 * time.Second}
	res, err := client.Get("http://127.0.0.1:" + cfg.Port + "/api/v1/health")
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned %d", res.StatusCode)
	}
	return nil
}

func serve(cfg config.Config, log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := store.CheckLocalFS(cfg.DataDir); err != nil {
		return err
	}
	if err := os.MkdirAll(cfg.UploadsDir(), 0o750); err != nil {
		return fmt.Errorf("can't create the data folder %s: %w", cfg.DataDir, err)
	}
	lock, err := store.Lock(cfg.DataDir)
	if err != nil {
		return err
	}
	defer lock.Close()

	st, err := store.Open(cfg.DBPath())
	if err != nil {
		return err
	}
	defer st.Close()

	err = st.Migrate(ctx, func() error {
		path, err := backup.Create(ctx, st, cfg.UploadsDir(), cfg.BackupDir, "pre-upgrade", time.Now())
		if err == nil {
			log.Info("safety backup created before upgrade", "file", path)
		}
		return err
	})
	if err != nil {
		return err
	}
	if err := st.IntegrityCheck(ctx); err != nil {
		return fmt.Errorf("%w. Stop here and restore the newest backup: wokko restore <file>", err)
	}

	svc := app.New(st, cfg, log)
	go svc.RunBackground(ctx)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           api.New(svc, log).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	errs := make(chan error, 1)
	go func() { errs <- srv.ListenAndServe() }()
	log.Info("Wokko is running", "port", cfg.Port, "version", version, "data", cfg.DataDir, "backups", cfg.BackupDir)

	select {
	case err := <-errs:
		return fmt.Errorf("web server: %w", err)
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdown); err != nil {
		return fmt.Errorf("shut down: %w", err)
	}
	return nil
}
