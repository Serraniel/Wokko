package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/serraniel/wokko/backend/internal/backup"
	"github.com/serraniel/wokko/backend/internal/store"
	"github.com/serraniel/wokko/backend/internal/store/sqlcgen"
)

const lowDiskBytes = 500 << 20

type checkResult struct {
	ran           bool
	err           error
	at            time.Time
	missingPhotos int64
}

// RunChecks verifies the database and that every recipe's photo exists on disk.
func (s *Service) RunChecks(ctx context.Context) {
	res := checkResult{ran: true, at: s.Now()}
	res.err = s.Store.IntegrityCheck(ctx)
	photos, err := s.Store.Queries.ListAllPhotos(ctx)
	if err != nil {
		res.err = errors.Join(res.err, err)
	}
	for _, p := range photos {
		if path, ok := s.Media.Path(p.String); ok {
			if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
				res.missingPhotos++
			}
		}
	}
	if res.err != nil {
		s.Log.Error("integrity check failed", "error", res.err)
	}
	if res.missingPhotos > 0 {
		s.Log.Warn("recipes point at photo files that don't exist", "count", res.missingPhotos)
	}
	s.healthMu.Lock()
	s.health = res
	s.healthMu.Unlock()
}

// RunBackup creates a backup and records the outcome, so a failure is visible to admins.
func (s *Service) RunBackup(ctx context.Context) (BackupRun, error) {
	s.backupMu.Lock()
	defer s.backupMu.Unlock()
	var id int64
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) (err error) {
		id, err = q.StartBackupRun(ctx, s.stamp(s.Now()))
		return err
	})
	if err != nil {
		return BackupRun{}, fmt.Errorf("record backup start: %w", err)
	}
	path, backupErr := backup.Create(ctx, s.Store, s.Cfg.UploadsDir(), s.Cfg.BackupDir, "", s.Now())
	p := sqlcgen.FinishBackupRunParams{FinishedAt: nullStr(s.stamp(s.Now())), Status: "ok", ID: id}
	if backupErr != nil {
		p.Status = "failed"
		p.Error = nullStr(backupErr.Error())
		s.Log.Error("backup failed", "error", backupErr)
	} else {
		p.File = nullStr(filepath.Base(path))
		if info, err := os.Stat(path); err == nil {
			p.SizeBytes = sql.NullInt64{Int64: info.Size(), Valid: true}
		}
		if err := backup.Prune(s.Cfg.BackupDir, s.Cfg.BackupRetentionDays, s.Now()); err != nil {
			s.Log.Warn("could not remove old backups", "error", err)
		}
		s.Log.Info("backup finished", "file", p.File.String, "bytes", p.SizeBytes.Int64)
	}
	if err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error { return q.FinishBackupRun(ctx, p) }); err != nil {
		return BackupRun{}, errors.Join(backupErr, fmt.Errorf("record backup result: %w", err))
	}
	return BackupRun{ID: id, Status: p.Status, File: p.File.String, SizeBytes: p.SizeBytes.Int64, Error: p.Error.String, FinishedAt: p.FinishedAt.String}, backupErr
}

// BackupDue reports whether a scheduled backup should run now. A failed attempt is retried
// hourly instead of waiting for the next full interval.
func (s *Service) BackupDue(ctx context.Context) (bool, error) {
	last, err := s.Store.Queries.LastBackupRun(ctx)
	if isNoRows(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	started, err := time.Parse(timeFormat, last.StartedAt)
	if err != nil {
		return true, nil
	}
	wait := s.Cfg.BackupInterval
	if last.Status != "ok" {
		wait = time.Hour
	}
	return s.Now().Sub(started) >= wait, nil
}

// RunBackground keeps backups, integrity checks and cleanup running until ctx ends.
func (s *Service) RunBackground(ctx context.Context) {
	s.RunChecks(ctx)
	tick := time.NewTicker(10 * time.Minute)
	defer tick.Stop()
	lastDaily := s.Now()
	for {
		due, err := s.BackupDue(ctx)
		if err != nil {
			s.Log.Error("could not check backup schedule", "error", err)
		} else if due {
			if _, err := s.RunBackup(ctx); err != nil {
				s.Log.Error("scheduled backup failed", "error", err)
			}
		}
		if s.Now().Sub(lastDaily) >= 24*time.Hour {
			lastDaily = s.Now()
			s.RunChecks(ctx)
			if err := s.PurgeExpiredTrash(ctx); err != nil {
				s.Log.Error("could not empty expired trash", "error", err)
			}
			err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error { return q.DeleteExpiredSessions(ctx, s.stamp(s.Now())) })
			if err != nil {
				s.Log.Error("could not remove expired sessions", "error", err)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

func toBackupRun(r sqlcgen.BackupRun) BackupRun {
	return BackupRun{ID: r.ID, StartedAt: r.StartedAt, FinishedAt: r.FinishedAt.String, Status: r.Status, File: r.File.String, SizeBytes: r.SizeBytes.Int64, Error: r.Error.String}
}

// GetHealth reports everything about data safety for the admin page.
func (s *Service) GetHealth(ctx context.Context, v Viewer) (Health, error) {
	if !v.isAdmin() {
		return Health{}, ErrForbidden
	}
	q := s.Store.Queries
	h := Health{Version: s.Cfg.Version, RecentBackups: []BackupRun{}}
	var err error
	if h.SchemaVersion, err = store.LatestSchemaVersion(); err != nil {
		return h, err
	}
	if h.Recipes, err = q.CountAllRecipes(ctx); err != nil {
		return h, err
	}
	if h.Foods, err = q.CountAllFoods(ctx); err != nil {
		return h, err
	}
	if h.Units, err = q.CountAllUnits(ctx); err != nil {
		return h, err
	}
	if h.Users, err = q.CountUsers(ctx); err != nil {
		return h, err
	}
	if last, err := q.LastBackupRun(ctx); err == nil {
		r := toBackupRun(last)
		h.LastBackup = &r
		h.BackupFailing = last.Status == "failed"
	} else if !isNoRows(err) {
		return h, err
	}
	h.BackupOverdue = true
	if ok, err := q.LastSuccessfulBackupRun(ctx); err == nil {
		r := toBackupRun(ok)
		h.LastSuccess = &r
		if t, err := time.Parse(timeFormat, ok.StartedAt); err == nil {
			h.BackupOverdue = s.Now().Sub(t) > 2*s.Cfg.BackupInterval
		}
	} else if !isNoRows(err) {
		return h, err
	}
	runs, err := q.ListBackupRuns(ctx, 10)
	if err != nil {
		return h, err
	}
	for _, r := range runs {
		h.RecentBackups = append(h.RecentBackups, toBackupRun(r))
	}
	if free, err := store.FreeBytes(s.Cfg.DataDir); err == nil {
		h.DataFreeBytes = int64(free)
		h.LowDisk = free < lowDiskBytes
	}
	if free, err := store.FreeBytes(s.Cfg.BackupDir); err == nil {
		h.BackupFreeBytes = int64(free)
		h.LowDisk = h.LowDisk || free < lowDiskBytes
	}
	s.healthMu.RLock()
	res := s.health
	s.healthMu.RUnlock()
	h.IntegrityOK = res.ran && res.err == nil
	if res.err != nil {
		h.IntegrityError = res.err.Error()
	}
	if res.ran {
		h.CheckedAt = s.stamp(res.at)
	}
	h.MissingPhotos = res.missingPhotos
	return h, nil
}
