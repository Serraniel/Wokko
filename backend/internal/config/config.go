// Package config reads settings from the environment. Wokko needs no secrets to start:
// there is no database password and sessions are random tokens stored hashed.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port                string
	DataDir             string
	BackupDir           string
	PublicURL           string
	BackupInterval      time.Duration
	BackupRetentionDays int
	TrashRetentionDays  int
	Version             string
	// SMTP sends mail when SMTPHost is set. Mail is optional: without it, links are copied.
	SMTP SMTP
}

// SMTP is the mail server Wokko hands its mail to: a provider, a sending service or a relay.
type SMTP struct {
	Host, Username, Password, From string
	Port                           int
	// TLS is starttls (port 587), tls (port 465) or none, for a relay on a private network.
	TLS string
}

func Load(version string) (Config, error) {
	c := Config{
		Port:      env("PORT", "8080"),
		DataDir:   env("DATA_DIR", "/data"),
		BackupDir: env("BACKUP_DIR", "/backups"),
		PublicURL: strings.TrimRight(env("PUBLIC_URL", ""), "/"),
		Version:   version,
	}
	hours, err := envInt("BACKUP_INTERVAL_HOURS", 24)
	if err != nil {
		return c, err
	}
	if hours < 1 {
		return c, fmt.Errorf("BACKUP_INTERVAL_HOURS must be at least 1, got %d", hours)
	}
	c.BackupInterval = time.Duration(hours) * time.Hour
	if c.BackupRetentionDays, err = envInt("BACKUP_RETENTION_DAYS", 30); err != nil {
		return c, err
	}
	if c.TrashRetentionDays, err = envInt("TRASH_RETENTION_DAYS", 30); err != nil {
		return c, err
	}
	c.SMTP = SMTP{Host: env("SMTP_HOST", ""), Username: env("SMTP_USERNAME", ""), Password: env("SMTP_PASSWORD", ""), From: env("SMTP_FROM", ""), TLS: strings.ToLower(env("SMTP_TLS", "starttls"))}
	if file := env("SMTP_PASSWORD_FILE", ""); file != "" {
		b, err := os.ReadFile(file)
		if err != nil {
			return c, fmt.Errorf("SMTP_PASSWORD_FILE: %w", err)
		}
		c.SMTP.Password = strings.TrimSpace(string(b))
	}
	port := 587
	if c.SMTP.TLS == "tls" {
		port = 465
	}
	if c.SMTP.Port, err = envInt("SMTP_PORT", port); err != nil {
		return c, err
	}
	if c.SMTP.Host != "" {
		if c.SMTP.TLS != "starttls" && c.SMTP.TLS != "tls" && c.SMTP.TLS != "none" {
			return c, fmt.Errorf("SMTP_TLS must be starttls, tls or none, got %q", c.SMTP.TLS)
		}
		if c.SMTP.From == "" {
			return c, fmt.Errorf("SMTP_FROM is needed when SMTP_HOST is set, such as Wokko <wokko@example.com>")
		}
	}
	return c, nil
}

func (c Config) DBPath() string     { return filepath.Join(c.DataDir, "wokko.db") }
func (c Config) UploadsDir() string { return filepath.Join(c.DataDir, "uploads") }

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func envInt(k string, d int) (int, error) {
	v := os.Getenv(k)
	if v == "" {
		return d, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number, got %q", k, v)
	}
	return n, nil
}
