// Package backup writes and restores Wokko backups: a verified database snapshot plus all
// uploaded photos in one tar.gz. Backups never stop the app.
package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/serraniel/wokko/backend/internal/store"
)

const (
	prefix    = "wokko-backup-"
	suffix    = ".tar.gz"
	dbEntry   = "wokko.db"
	uploadDir = "uploads"
	// KeepAtLeast backups always survive retention, even if backups stopped for a long time.
	KeepAtLeast = 3
)

// Create writes a backup into backupDir and returns its path. The file only gets its final
// name after everything was written and the database copy passed its integrity check.
func Create(ctx context.Context, s *store.Store, uploads, backupDir, label string, now time.Time) (string, error) {
	if err := os.MkdirAll(backupDir, 0o750); err != nil {
		return "", fmt.Errorf("backup folder %s: %w", backupDir, err)
	}
	tmpDir, err := os.MkdirTemp(filepath.Dir(s.Path()), "backup-")
	if err != nil {
		return "", fmt.Errorf("temporary folder: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	snap := filepath.Join(tmpDir, dbEntry)
	if err := s.SnapshotTo(ctx, snap); err != nil {
		return "", err
	}
	if err := verifyDB(ctx, snap); err != nil {
		return "", fmt.Errorf("snapshot failed verification: %w", err)
	}

	name := prefix + now.UTC().Format("2006-01-02T15-04-05Z")
	if label != "" {
		name += "-" + label
	}
	final := filepath.Join(backupDir, name+suffix)
	tmp := final + ".partial"
	if err := writeArchive(tmp, snap, uploads); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := Verify(tmp); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("archive failed verification: %w", err)
	}
	if err := os.Rename(tmp, final); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("finish backup: %w", err)
	}
	return final, nil
}

func writeArchive(path, dbFile, uploads string) (err error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return fmt.Errorf("create backup file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); err == nil && cerr != nil {
			err = cerr
		}
	}()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	if err := addFile(tw, dbFile, dbEntry); err != nil {
		return err
	}
	entries, err := os.ReadDir(uploads)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read uploads: %w", err)
	}
	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}
		if err := addFile(tw, filepath.Join(uploads, e.Name()), uploadDir+"/"+e.Name()); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("finish archive: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("finish archive: %w", err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("flush backup to disk: %w", err)
	}
	return nil
}

func addFile(tw *tar.Writer, path, name string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("add %s to backup: %w", name, err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("add %s to backup: %w", name, err)
	}
	hdr := &tar.Header{Name: name, Mode: 0o640, Size: info.Size(), ModTime: info.ModTime(), Typeflag: tar.TypeReg}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("add %s to backup: %w", name, err)
	}
	if _, err := io.Copy(tw, f); err != nil {
		return fmt.Errorf("add %s to backup: %w", name, err)
	}
	return nil
}

func verifyDB(ctx context.Context, path string) error {
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return err
	}
	defer db.Close()
	return store.CheckDB(ctx, db)
}

// Verify unpacks a backup to a temporary folder and checks the database inside it.
func Verify(archive string) error {
	tmp, err := os.MkdirTemp(filepath.Dir(archive), ".verify-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if err := extract(archive, tmp); err != nil {
		return err
	}
	return verifyDB(context.Background(), filepath.Join(tmp, dbEntry))
}

func extract(archive, dest string) error {
	f, err := os.Open(archive)
	if err != nil {
		return fmt.Errorf("open backup: %w", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("read backup: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	hasDB := false
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read backup: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		clean := filepath.Clean(hdr.Name)
		ok := clean == dbEntry || (strings.HasPrefix(clean, uploadDir+string(filepath.Separator)) && filepath.Base(clean) != "..")
		if !ok || filepath.IsAbs(clean) || strings.Contains(clean, "..") {
			return fmt.Errorf("backup contains an unexpected file: %q", hdr.Name)
		}
		target := filepath.Join(dest, clean)
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
		if err != nil {
			return err
		}
		_, cerr := io.Copy(out, tr)
		if err := errors.Join(cerr, out.Close()); err != nil {
			return fmt.Errorf("unpack %s: %w", hdr.Name, err)
		}
		hasDB = hasDB || clean == dbEntry
	}
	if !hasDB {
		return errors.New("backup has no database file")
	}
	return nil
}

// Restore replaces the data directory's contents with a backup. It verifies the backup first
// and moves the current data aside, never deleting it. The app must not be running.
func Restore(archive, dataDir string, now time.Time) (keptAt string, err error) {
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		return "", err
	}
	lock, err := store.Lock(dataDir)
	if err != nil {
		return "", fmt.Errorf("stop Wokko before restoring: %w", err)
	}
	defer lock.Close()

	staging := filepath.Join(dataDir, ".restore-staging")
	if err := os.RemoveAll(staging); err != nil {
		return "", err
	}
	defer os.RemoveAll(staging)
	if err := os.MkdirAll(staging, 0o750); err != nil {
		return "", err
	}
	if err := extract(archive, staging); err != nil {
		return "", err
	}
	if err := verifyDB(context.Background(), filepath.Join(staging, dbEntry)); err != nil {
		return "", fmt.Errorf("backup failed verification, nothing was changed: %w", err)
	}

	keptAt = filepath.Join(dataDir, "pre-restore-"+now.UTC().Format("2006-01-02T15-04-05Z"))
	if err := os.MkdirAll(keptAt, 0o750); err != nil {
		return "", err
	}
	for _, n := range []string{dbEntry, dbEntry + "-wal", dbEntry + "-shm", uploadDir} {
		src := filepath.Join(dataDir, n)
		if err := os.Rename(src, filepath.Join(keptAt, n)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return keptAt, fmt.Errorf("move current data aside: %w", err)
		}
	}
	if err := os.Rename(filepath.Join(staging, dbEntry), filepath.Join(dataDir, dbEntry)); err != nil {
		return keptAt, err
	}
	if err := os.Rename(filepath.Join(staging, uploadDir), filepath.Join(dataDir, uploadDir)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return keptAt, err
	}
	return keptAt, nil
}

// Prune deletes backups older than retention, always keeping the newest KeepAtLeast files.
func Prune(backupDir string, retentionDays int, now time.Time) error {
	if retentionDays <= 0 {
		return nil
	}
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix) && strings.HasSuffix(e.Name(), suffix) {
			names = append(names, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names))) // the timestamp in the name sorts by age
	cutoff := now.AddDate(0, 0, -retentionDays)
	var errs []error
	for i, n := range names {
		if i < KeepAtLeast {
			continue
		}
		info, err := os.Stat(filepath.Join(backupDir, n))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if info.ModTime().Before(cutoff) {
			errs = append(errs, os.Remove(filepath.Join(backupDir, n)))
		}
	}
	return errors.Join(errs...)
}
