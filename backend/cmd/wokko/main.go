// Command wokko runs the Wokko server. It also has a restore command for backups.
package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
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
		case len(os.Args) > 1 && os.Args[1] == "mcp-stdio":
			err = mcpStdio(cfg, os.Args[2:], os.Stdin, os.Stdout)
		case len(os.Args) > 1 && os.Args[1] == "version":
			fmt.Println(version)
		case len(os.Args) > 1:
			err = fmt.Errorf("unknown command %q. Commands: restore <backup file>, mcp-stdio --token <token> [--url <address>], healthcheck, version", os.Args[1])
		default:
			err = serve(cfg, log)
		}
	}
	if err != nil {
		log.Error("Wokko stopped", "reason", err)
		os.Exit(1)
	}
}

// mcpStdio lets an assistant that only speaks MCP over stdin and stdout reach a running
// Wokko. It holds no data itself. Each line goes to the server's /mcp, and the answer comes back.
func mcpStdio(cfg config.Config, args []string, in io.Reader, out io.Writer) error {
	flags := flag.NewFlagSet("mcp-stdio", flag.ContinueOnError)
	token := flags.String("token", os.Getenv("WOKKO_TOKEN"), "access token from Wokko's settings, or WOKKO_TOKEN")
	address := flags.String("url", "http://127.0.0.1:"+cfg.Port+"/mcp", "address of Wokko's /mcp")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *token == "" {
		return errors.New("usage: wokko mcp-stdio --token <token> [--url <address>]")
	}
	client := &http.Client{Timeout: time.Minute}
	lines := bufio.NewScanner(in)
	lines.Buffer(make([]byte, 1<<20), 16<<20)
	for lines.Scan() {
		req, err := http.NewRequest(http.MethodPost, *address, bytes.NewReader(lines.Bytes()))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+*token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		res, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("can't reach Wokko at %s: %w", *address, err)
		}
		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return err
		}
		switch {
		case res.StatusCode == http.StatusUnauthorized:
			return errors.New("Wokko didn't accept this token")
		case res.StatusCode == http.StatusNotFound:
			return errors.New("MCP is switched off in Wokko's admin settings")
		case res.StatusCode >= 300:
			return fmt.Errorf("Wokko answered with status %d", res.StatusCode)
		}
		if len(bytes.TrimSpace(body)) > 0 {
			if _, err := fmt.Fprintf(out, "%s\n", bytes.TrimSpace(body)); err != nil {
				return err
			}
		}
	}
	return lines.Err()
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
	// A newer starter catalog reaches this database here. It runs in one transaction and never
	// deletes, so a failure only means the update waits for the next start.
	if _, err := svc.SyncCatalog(ctx); err != nil {
		log.Error("starter catalog update failed, it will be tried again on the next start", "error", err)
	}
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
