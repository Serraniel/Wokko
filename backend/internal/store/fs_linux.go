package store

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

var networkFS = map[int64]string{
	0x6969:     "NFS",
	0xFF534D42: "CIFS/SMB",
	0xFE534D42: "SMB2",
	0x517B:     "SMB",
	0x01021997: "9P",
	0x5346414F: "AFS",
}

// CheckLocalFS refuses network filesystems: SQLite's locking is unreliable there and
// this is the classic way to corrupt a database. Backups belong on the NAS, data doesn't.
//
// dir doesn't have to exist yet: its nearest existing parent decides, so nothing gets created
// on a share just to find out it's a share.
func CheckLocalFS(dir string) error {
	existing := filepath.Clean(dir)
	for {
		if _, err := os.Stat(existing); err == nil || existing == filepath.Dir(existing) {
			break
		}
		existing = filepath.Dir(existing)
	}
	var st syscall.Statfs_t
	if err := syscall.Statfs(existing, &st); err != nil {
		return fmt.Errorf("check filesystem of %s: %w", dir, err)
	}
	if name, bad := networkFS[int64(uint32(st.Type))]; bad {
		return fmt.Errorf("%s is on a network filesystem (%s). Wokko's database must be on a local disk "+
			"because a network share can corrupt it. Point DATA_DIR at local storage and use BACKUP_DIR for the NAS", dir, name)
	}
	return nil
}

// FreeBytes returns the space available to this process on dir's filesystem.
func FreeBytes(dir string) (uint64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return 0, err
	}
	return st.Bavail * uint64(st.Bsize), nil
}

// Lock takes an exclusive lock on the data directory so two processes can't use one database.
// The lock is released when the process exits.
func Lock(dir string) (*os.File, error) {
	f, err := os.OpenFile(filepath.Join(dir, "wokko.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("data directory %s is already in use by another Wokko process", dir)
	}
	return f, nil
}
