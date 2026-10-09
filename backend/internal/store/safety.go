package store

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// PreMigrationKeep is how many "before upgrading" copies are kept.
const PreMigrationKeep = 5

// pendingMigrations counts the migrations in the binary that the database has not applied yet, and whether the database already
// holds a game (a schema recorded and a player).
func pendingMigrations(db *sql.DB) (pending int, hasGame bool, err error) {
	var tables int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name IN ('schema_migrations', 'players')`).Scan(&tables); err != nil {
		return 0, false, err
	}
	applied := map[int]bool{}
	if tables == 2 {
		rows, err := db.Query(`SELECT version FROM schema_migrations`)
		if err != nil {
			return 0, false, err
		}
		for rows.Next() {
			var v int
			rows.Scan(&v)
			applied[v] = true
		}
		rows.Close()
		var players int
		db.QueryRow(`SELECT COUNT(*) FROM players`).Scan(&players)
		hasGame = players > 0
	}
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return 0, false, err
	}
	for _, e := range entries {
		v, err := strconv.Atoi(strings.SplitN(e.Name(), "_", 2)[0])
		if err == nil && !applied[v] {
			pending++
		}
	}
	return pending, hasGame, nil
}

// OpenWithSafety is Open for the real server: if the database holds a game and the new version of the app has migrations to
// apply, a verified copy of the database as it was is written to dir/pre-migration BEFORE the first migration runs. An upgrade
// that goes wrong can then always be undone with deploy/restore.sh. A fresh database, or one that is already up to date, gets no copy.
func OpenWithSafety(path, backupDir string, now time.Time, logf func(string, ...any)) (*sql.DB, error) {
	dsn := "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_txlock=immediate"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	pending, hasGame, err := pendingMigrations(db)
	if err != nil {
		db.Close()
		return nil, err
	}
	if pending > 0 && hasGame && backupDir != "" {
		from := 0
		db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&from)
		dir := filepath.Join(backupDir, "pre-migration")
		name, err := preMigrationBackup(db, dir, from, now)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("could not take the safety copy before upgrading, so the upgrade did not start: %w", err)
		}
		if logf != nil {
			logf("upgrade: %d migration(s) pending, safety copy %s", pending, name)
		}
	}
	if err := Migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func preMigrationBackup(db *sql.DB, dir string, fromVersion int, now time.Time) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("pomofarm-v%d-%s.db", fromVersion, now.UTC().Format("20060102-150405"))
	path := filepath.Join(dir, name)
	tmp := path + ".tmp"
	os.Remove(tmp)
	if _, err := db.Exec(`VACUUM INTO ?`, tmp); err != nil {
		os.Remove(tmp)
		return "", err
	}
	if err := verifyFile(tmp); err != nil {
		os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", err
	}
	// keep the newest few
	entries, _ := os.ReadDir(dir)
	var names []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "pomofarm-v") && strings.HasSuffix(e.Name(), ".db") {
			names = append(names, e.Name())
		}
	}
	sortByModTime(dir, names)
	for len(names) > PreMigrationKeep {
		os.Remove(filepath.Join(dir, names[0]))
		names = names[1:]
	}
	return path, nil
}

// sortByModTime orders names oldest first by file modification time (the timestamps in these names do not sort by version).
func sortByModTime(dir string, names []string) {
	mod := func(n string) time.Time {
		i, err := os.Stat(filepath.Join(dir, n))
		if err != nil {
			return time.Time{}
		}
		return i.ModTime()
	}
	for i := 1; i < len(names); i++ {
		for j := i; j > 0 && mod(names[j]).Before(mod(names[j-1])); j-- {
			names[j], names[j-1] = names[j-1], names[j]
		}
	}
}
