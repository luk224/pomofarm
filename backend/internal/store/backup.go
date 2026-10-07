package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	backupPrefix = "pomofarm-"
	backupSuffix = ".db"
)

// Backup writes a consistent copy of the database into dir using VACUUM INTO
// and then keeps only the newest `keep` backups. It returns the new file path.
func Backup(db *sql.DB, dir string, keep int, now time.Time) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := backupPrefix + now.UTC().Format("20060102-150405") + backupSuffix
	path := filepath.Join(dir, name)
	tmp := path + ".tmp"
	os.Remove(tmp)
	if _, err := db.Exec(`VACUUM INTO ?`, tmp); err != nil {
		return "", fmt.Errorf("vacuum into: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", err
	}
	return path, prune(dir, keep)
}

// ListBackups returns the backup file names in dir, oldest first.
func ListBackups(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		n := e.Name()
		if strings.HasPrefix(n, backupPrefix) && strings.HasSuffix(n, backupSuffix) {
			names = append(names, n)
		}
	}
	sort.Strings(names) // timestamp in the name: lexical order = chronological
	return names, nil
}

func prune(dir string, keep int) error {
	names, err := ListBackups(dir)
	if err != nil {
		return err
	}
	for len(names) > keep {
		if err := os.Remove(filepath.Join(dir, names[0])); err != nil {
			return err
		}
		names = names[1:]
	}
	return nil
}

// LatestBackupTime returns the time encoded in the newest backup name (zero if none).
func LatestBackupTime(dir string) (time.Time, error) {
	names, err := ListBackups(dir)
	if err != nil || len(names) == 0 {
		return time.Time{}, err
	}
	n := strings.TrimSuffix(strings.TrimPrefix(names[len(names)-1], backupPrefix), backupSuffix)
	return time.Parse("20060102-150405", n)
}

// RunBackups takes a backup when the newest one is older than every, checking
// once per check interval, until stop is closed. Errors are reported via logf.
func RunBackups(db *sql.DB, dir string, keep int, every, check time.Duration, stop <-chan struct{}, logf func(string, ...any)) {
	tick := func() {
		last, err := LatestBackupTime(dir)
		if err != nil {
			logf("backup: %v", err)
			return
		}
		if time.Since(last) < every {
			return
		}
		if p, err := Backup(db, dir, keep, time.Now()); err != nil {
			logf("backup: %v", err)
		} else {
			logf("backup: wrote %s", p)
		}
	}
	tick()
	t := time.NewTicker(check)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			tick()
		case <-stop:
			return
		}
	}
}
