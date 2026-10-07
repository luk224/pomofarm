package store

import (
	"path/filepath"
	"strings"
	"testing"
)

func tempPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "test.db")
}

func TestMigrateEmptyDBAndIdempotent(t *testing.T) {
	path := tempPath(t)
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	v, err := SchemaVersion(db)
	if err != nil || v != 3 {
		t.Fatalf("version = %d, err = %v; want 3", v, err)
	}
	var mode string
	db.QueryRow(`PRAGMA journal_mode`).Scan(&mode)
	if mode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", mode)
	}
	db.Close()

	db, err = Open(path) // second open must not re-apply
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&n)
	if n != 3 {
		t.Fatalf("schema_migrations rows = %d, want 3", n)
	}
}

func TestOnlyOneActivePomodoroPerPlayer(t *testing.T) {
	db, err := Open(tempPath(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	exec := func(q string, args ...any) error { _, err := db.Exec(q, args...); return err }

	if err := exec(`INSERT INTO players (id,name,created_at,last_seen_at) VALUES (1,'a','t','t')`); err != nil {
		t.Fatal(err)
	}
	ins := `INSERT INTO pomodoros (player_id,plant_type,planned_s,started_at,status) VALUES (1,'daisy',600,'t',?)`
	if err := exec(ins, "running"); err != nil {
		t.Fatal(err)
	}
	err = exec(ins, "paused")
	if err == nil || !strings.Contains(err.Error(), "UNIQUE") {
		t.Fatalf("second active pomodoro: err = %v, want UNIQUE violation", err)
	}
	// finished ones don't count
	if err := exec(ins, "completed"); err != nil {
		t.Fatalf("completed pomodoro rejected: %v", err)
	}
	if err := exec(ins, "cancelled"); err != nil {
		t.Fatalf("cancelled pomodoro rejected: %v", err)
	}
}

func TestForeignKeysEnforced(t *testing.T) {
	db, err := Open(tempPath(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`INSERT INTO plots (player_id,x,y) VALUES (99,0,0)`); err == nil {
		t.Fatal("plot with unknown player accepted; foreign_keys is off")
	}
}
