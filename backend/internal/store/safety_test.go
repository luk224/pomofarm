package store

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func fileHash(t *testing.T, path string) [32]byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return sha256.Sum256(b)
}

func populated(t *testing.T) (*sql.DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "game.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`INSERT INTO players (id,name,coins_milli,created_at,last_seen_at) VALUES (1,'luk',777,'t','t')`); err != nil {
		t.Fatal(err)
	}
	return db, path
}

// ---------- Verify ----------

func TestVerifyAcceptsAHealthyDatabaseAndTouchesNothing(t *testing.T) {
	db, _ := populated(t)
	dir := t.TempDir()
	backup, err := Backup(db, dir, 14, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	before := fileHash(t, backup)
	if err := Verify(backup); err != nil {
		t.Fatalf("a fresh backup must verify: %v", err)
	}
	if fileHash(t, backup) != before {
		t.Fatal("verifying must never write to the file")
	}
	if names, _ := ListBackups(dir); len(names) != 1 {
		t.Fatalf("%d files in the backup directory", len(names))
	}
}

func TestVerifyRejectsEverythingThatIsNotAHealthyPomoFarmDatabase(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, data []byte) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, data, 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	// a real database to damage
	db, good := populated(t)
	if _, err := Backup(db, filepath.Dir(good), 5, time.Now()); err != nil {
		t.Fatal(err)
	}
	names, _ := ListBackups(filepath.Dir(good))
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(good), names[0]))
	if err != nil {
		t.Fatal(err)
	}
	// a SQLite file that is not ours
	other := filepath.Join(dir, "other.db")
	o, _ := sql.Open("sqlite", "file:"+other)
	o.Exec(`CREATE TABLE notes (id INTEGER)`)
	o.Close()
	// ours, but with the migrations table emptied
	noMig := filepath.Join(dir, "nomig.db")
	if err := os.WriteFile(noMig, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	n, _ := sql.Open("sqlite", "file:"+noMig)
	n.Exec(`DELETE FROM schema_migrations`)
	n.Close()

	flipped := append([]byte(nil), raw...)
	for i := len(flipped) / 2; i < len(flipped)/2+4000 && i < len(flipped); i++ {
		flipped[i] ^= 0xFF
	}
	cases := map[string]string{
		"missing":                   filepath.Join(dir, "nope.db"),
		"empty":                     write("empty.db", nil),
		"random bytes":              write("random.db", []byte("this is not a database, just some text that is long enough to look like one")),
		"truncated":                 write("cut.db", raw[:len(raw)/3]),
		"damaged pages":             write("flipped.db", flipped),
		"another SQLite database":   other,
		"migrations table is empty": noMig,
		"a directory":               dir,
	}
	for name, path := range cases {
		if err := Verify(path); err == nil {
			t.Errorf("%s: Verify accepted it", name)
		}
	}
}

// ---------- Backup never keeps (or prunes for) a bad copy ----------

func TestABackupThatFailsVerificationIsDiscardedAndOldOnesSurvive(t *testing.T) {
	db, _ := populated(t)
	dir := t.TempDir()
	start := time.Date(2026, 1, 1, 3, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		if _, err := Backup(db, dir, 3, start.AddDate(0, 0, i)); err != nil {
			t.Fatal(err)
		}
	}
	saved := verifyFile
	defer func() { verifyFile = saved }()
	verifyFile = func(string) error { return errors.New("simulated damage") }
	if _, err := Backup(db, dir, 3, start.AddDate(0, 0, 3)); err == nil {
		t.Fatal("a backup that does not verify must be reported as an error")
	}
	names, _ := ListBackups(dir)
	if len(names) != 3 || names[0] != "pomofarm-20260101-030000.db" {
		t.Fatalf("the three good backups must all survive (the oldest included), got %v", names)
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Fatalf("a temporary file was left behind: %s", e.Name())
		}
	}
}

// ---------- the safety copy before an upgrade ----------

// oldDatabase builds a game as an older version of the app would have left it (migrations 1..through only).
func oldDatabase(t *testing.T, through int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "old.db")
	raw, err := sql.Open("sqlite", "file:"+path+"?_pragma=journal_mode(WAL)")
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	if _, err := raw.Exec(`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	for v, name := range []string{"0001_init.sql", "0002_silo.sql", "0003_rest.sql", "0004_structures.sql", "0005_decor.sql"} {
		if v+1 > through {
			break
		}
		body, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := raw.Exec(string(body)); err != nil {
			t.Fatal(err)
		}
		raw.Exec(`INSERT INTO schema_migrations VALUES (?, 't')`, v+1)
	}
	if _, err := raw.Exec(`INSERT INTO players (id,name,coins_milli,created_at,last_seen_at) VALUES (1,'Luk',4242,'t','t')`); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestUpgradingAGameFirstTakesAVerifiedSafetyCopyOfTheOldDatabase(t *testing.T) {
	path := oldDatabase(t, 5)
	backups := t.TempDir()
	var logged []string
	db, err := OpenWithSafety(path, backups, time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC), func(f string, a ...any) { logged = append(logged, f) })
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if v, _ := SchemaVersion(db); v != 6 {
		t.Fatalf("the game was upgraded to %d", v)
	}
	files, _ := filepath.Glob(filepath.Join(backups, "pre-migration", "*.db"))
	if len(files) != 1 || filepath.Base(files[0]) != "pomofarm-v5-20261009-120000.db" {
		t.Fatalf("expected one safety copy named for the old version, got %v", files)
	}
	if err := Verify(files[0]); err != nil {
		t.Fatalf("the safety copy must be a valid database: %v", err)
	}
	// it holds the game exactly as it was BEFORE the upgrade
	old, _ := sql.Open("sqlite", "file:"+files[0]+"?mode=ro")
	defer old.Close()
	var coins int64
	var version int
	old.QueryRow(`SELECT coins_milli FROM players WHERE id=1`).Scan(&coins)
	old.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&version)
	if coins != 4242 || version != 5 {
		t.Fatalf("the safety copy has coins %d at schema %d, want 4242 at 5", coins, version)
	}
	if len(logged) == 0 {
		t.Fatal("the upgrade should be logged")
	}
	// the live database kept the data too
	var live int64
	db.QueryRow(`SELECT coins_milli FROM players WHERE id=1`).Scan(&live)
	if live != 4242 {
		t.Fatalf("live coins %d", live)
	}
}

func TestNoSafetyCopyWhenThereIsNothingToUpgrade(t *testing.T) {
	backups := t.TempDir()
	// a brand-new install
	fresh := filepath.Join(t.TempDir(), "fresh.db")
	db, err := OpenWithSafety(fresh, backups, time.Now(), nil)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	// an up-to-date game, opened again
	game := oldDatabase(t, 5)
	d1, err := OpenWithSafety(game, t.TempDir(), time.Now(), nil)
	if err != nil {
		t.Fatal(err)
	}
	d1.Close()
	d2, err := OpenWithSafety(game, backups, time.Now(), nil)
	if err != nil {
		t.Fatal(err)
	}
	d2.Close()
	if files, _ := filepath.Glob(filepath.Join(backups, "pre-migration", "*")); len(files) != 0 {
		t.Fatalf("no upgrade, no safety copy: %v", files)
	}
}

func TestIfTheSafetyCopyCannotBeMadeTheUpgradeDoesNotStart(t *testing.T) {
	path := oldDatabase(t, 5)
	saved := verifyFile
	defer func() { verifyFile = saved }()
	verifyFile = func(string) error { return errors.New("disk full, say") }
	if _, err := OpenWithSafety(path, t.TempDir(), time.Now(), nil); err == nil {
		t.Fatal("without a safety copy the upgrade must not run")
	}
	raw, _ := sql.Open("sqlite", "file:"+path+"?mode=ro")
	defer raw.Close()
	var v int
	raw.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&v)
	if v != 5 {
		t.Fatalf("the database was migrated to %d even though the safety copy failed", v)
	}
}

func TestOnlyTheNewestSafetyCopiesAreKept(t *testing.T) {
	backups := t.TempDir()
	for i := 0; i < PreMigrationKeep+3; i++ {
		path := oldDatabase(t, 5)
		db, err := OpenWithSafety(path, backups, time.Date(2026, 10, 9, 12, 0, i, 0, time.UTC), nil)
		if err != nil {
			t.Fatal(err)
		}
		db.Close()
		os.Chtimes(filepath.Join(backups, "pre-migration", "pomofarm-v5-"+time.Date(2026, 10, 9, 12, 0, i, 0, time.UTC).Format("20060102-150405")+".db"), time.Now().Add(time.Duration(i)*time.Minute), time.Now().Add(time.Duration(i)*time.Minute))
	}
	files, _ := filepath.Glob(filepath.Join(backups, "pre-migration", "*.db"))
	if len(files) != PreMigrationKeep {
		t.Fatalf("kept %d safety copies, want %d", len(files), PreMigrationKeep)
	}
}

func TestSafetyCopiesDoNotDisturbTheDailyBackups(t *testing.T) {
	path := oldDatabase(t, 5)
	backups := t.TempDir()
	db, err := OpenWithSafety(path, backups, time.Now(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if names, _ := ListBackups(backups); len(names) != 0 {
		t.Fatalf("a safety copy must not look like a daily backup: %v", names)
	}
	if _, err := LatestBackupTime(backups); err != nil {
		t.Fatal(err)
	}
}
