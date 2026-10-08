package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestBackupIsRestorableAndPrunes(t *testing.T) {
	db, err := Open(tempPath(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`INSERT INTO players (id,name,coins_milli,created_at,last_seen_at) VALUES (1,'luk',12345,'t','t')`); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	start := time.Date(2026, 1, 1, 3, 0, 0, 0, time.UTC)
	var last string
	for i := 0; i < 16; i++ {
		last, err = Backup(db, dir, 14, start.AddDate(0, 0, i))
		if err != nil {
			t.Fatal(err)
		}
	}
	names, _ := ListBackups(dir)
	if len(names) != 14 {
		t.Fatalf("kept %d backups, want 14", len(names))
	}
	if names[0] != "pomofarm-20260103-030000.db" {
		t.Fatalf("oldest kept = %s, want the 3rd day", names[0])
	}
	if filepath.Base(last) != names[13] {
		t.Fatalf("newest = %s, want %s", names[13], filepath.Base(last))
	}
	// the backup opens as a normal, migrated, populated database
	restored, err := Open(last)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	var coins int64
	if err := restored.QueryRow(`SELECT coins_milli FROM players WHERE id=1`).Scan(&coins); err != nil || coins != 12345 {
		t.Fatalf("restored coins = %d, err = %v", coins, err)
	}
	if v, _ := SchemaVersion(restored); v != 6 {
		t.Fatalf("restored schema version = %d", v)
	}
	if lt, _ := LatestBackupTime(dir); !lt.Equal(start.AddDate(0, 0, 15)) {
		t.Fatalf("latest backup time = %v", lt)
	}
}
