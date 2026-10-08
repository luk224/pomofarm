package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// A database created by the Phase 1 app (schema 1) holds a real player's data. Upgrading the app, or restoring an old
// backup into a newer app, must apply migrations 2 and 3 without losing or altering a single row.
func TestUpgradingAnOldDatabaseKeepsEveryRow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	raw, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	first, err := migrationsFS.ReadFile("migrations/0001_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		string(first),
		`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`,
		`INSERT INTO schema_migrations VALUES (1, '2026-10-07T12:00:00Z')`,
		`INSERT INTO players (id,name,focus_points,lifetime_focus,coins_milli,silo_level,season,biome,created_at,last_seen_at,version)
			VALUES (1,'Luk',37,120,4200,2,3,'spring','2026-10-01T08:00:00Z','2026-10-07T11:00:00Z',9)`,
		`INSERT INTO plots (id,player_id,x,y,plant_type,state,planted_at,grow_s,matured_at,harvested,life_s,wilts_at,collected_to,version)
			VALUES (1,1,1,1,'apple','mature','2026-10-06T08:00:00Z',2700,'2026-10-06T08:45:00Z',1,259200,'2026-10-09T08:45:00Z','2026-10-07T10:00:00Z',4),
			       (2,1,2,1,NULL,'empty',NULL,NULL,NULL,0,NULL,NULL,NULL,0)`,
		`INSERT INTO unlocks VALUES (1,'seed','daisy','2026-10-01T08:00:00Z'),(1,'seed','apple','2026-10-04T08:00:00Z')`,
		`INSERT INTO tags (id,player_id,name) VALUES (1,1,'tesis')`,
		`INSERT INTO pomodoros (id,player_id,plot_id,plant_type,tag_id,planned_s,started_at,paused_total_s,ended_at,reward_focus,status)
			VALUES (1,1,1,'apple',1,2700,'2026-10-06T08:00:00Z',60,'2026-10-06T08:46:00Z',15,'completed')`,
		`INSERT INTO pomodoro_events (pomodoro_id,kind,at) VALUES (1,'start','2026-10-06T08:00:00Z'),(1,'complete','2026-10-06T08:46:00Z')`,
		`INSERT INTO settings VALUES (1,'tutorial_done','1')`,
	} {
		if _, err := raw.Exec(q); err != nil {
			t.Fatalf("setting up the old database: %v\n%s", err, q)
		}
	}
	raw.Close()

	db, err := Open(path) // the app starting after an upgrade
	if err != nil {
		t.Fatalf("opening an old database: %v", err)
	}
	defer db.Close()
	if v, _ := SchemaVersion(db); v != 6 {
		t.Fatalf("schema version = %d, want 6", v)
	}

	var name string
	var focus, lifetime, coins, silo, season, version, siloMicro, peak int64
	var restStart, restUntil sql.NullString
	if err := db.QueryRow(`SELECT name,focus_points,lifetime_focus,coins_milli,silo_level,season,version,silo_micro,silo_peak_micro_h,rest_started_at,rest_until FROM players WHERE id=1`).
		Scan(&name, &focus, &lifetime, &coins, &silo, &season, &version, &siloMicro, &peak, &restStart, &restUntil); err != nil {
		t.Fatal(err)
	}
	if name != "Luk" || focus != 37 || lifetime != 120 || coins != 4200 || silo != 2 || season != 3 || version != 9 {
		t.Fatalf("player changed by the upgrade: %s %d %d %d %d %d %d", name, focus, lifetime, coins, silo, season, version)
	}
	if siloMicro != 0 || peak != 0 || restStart.Valid || restUntil.Valid {
		t.Fatalf("new columns should start empty: silo %d peak %d rest %v %v", siloMicro, peak, restStart, restUntil)
	}
	for table, want := range map[string]int{"plots": 2, "unlocks": 2, "tags": 1, "pomodoros": 1, "pomodoro_events": 2, "settings": 1} {
		var n int
		db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n)
		if n != want {
			t.Errorf("%s has %d rows after the upgrade, want %d", table, n, want)
		}
	}
	var plant, state, collected string
	if err := db.QueryRow(`SELECT plant_type,state,collected_to FROM plots WHERE id=1`).Scan(&plant, &state, &collected); err != nil || plant != "apple" || state != "mature" || collected != "2026-10-07T10:00:00Z" {
		t.Fatalf("plot 1: %v %v %v err=%v", plant, state, collected, err)
	}
	// the partial index of one active pomodoro still works after the upgrade
	if _, err := db.Exec(`INSERT INTO pomodoros (player_id,plant_type,planned_s,started_at,status) VALUES (1,'daisy',600,'t','running')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO pomodoros (player_id,plant_type,planned_s,started_at,status) VALUES (1,'daisy',600,'t','paused')`); err == nil {
		t.Fatal("a second active pomodoro was accepted after the upgrade")
	}
}

func TestMigrationsApplyOnceAndInOrderEvenFromHalfwayStates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	for i := 0; i < 3; i++ { // opening again and again is harmless
		db, err = Open(path)
		if err != nil {
			t.Fatal(err)
		}
		var n int
		db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&n)
		db.Close()
		if n != 6 {
			t.Fatalf("open #%d: %d migrations recorded, want 6", i+1, n)
		}
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

// Phase 3: a database at schema 4 (hives and the Dog already bought) must come through migration 5 untouched, and the
// new decoration rules must then hold on it.
func TestUpgradingFromSchema4KeepsStructures(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v4.db")
	raw, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	for v, name := range []string{"0001_init.sql", "0002_silo.sql", "0003_rest.sql", "0004_structures.sql"} {
		body, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := raw.Exec(string(body)); err != nil {
			t.Fatalf("applying %s: %v", name, err)
		}
		if _, err := raw.Exec(`INSERT INTO schema_migrations VALUES (?, '2026-10-07T12:00:00Z')`, v+1); err != nil {
			t.Fatal(err)
		}
	}
	for _, q := range []string{
		`INSERT INTO players (id,name,focus_points,lifetime_focus,coins_milli,silo_level,season,biome,created_at,last_seen_at,version)
			VALUES (1,'Luk',500,900,12000000,3,2,'spring','2026-10-01T08:00:00Z','2026-10-07T11:00:00Z',7)`,
		`INSERT INTO structures (id,player_id,kind,x,y) VALUES (1,1,'hive',1,1),(2,1,'hive',2,2),(3,1,'dog',0,0)`,
	} {
		if _, err := raw.Exec(q); err != nil {
			t.Fatalf("setting up the v4 database: %v", err)
		}
	}
	raw.Close()

	db, err := Open(path)
	if err != nil {
		t.Fatalf("opening a schema-4 database: %v", err)
	}
	defer db.Close()
	if v, _ := SchemaVersion(db); v != 6 {
		t.Fatalf("schema version = %d, want 6", v)
	}
	var hives, dogs int
	db.QueryRow(`SELECT COUNT(*) FROM structures WHERE kind='hive'`).Scan(&hives)
	db.QueryRow(`SELECT COUNT(*) FROM structures WHERE kind='dog'`).Scan(&dogs)
	var coins int64
	db.QueryRow(`SELECT coins_milli FROM players WHERE id=1`).Scan(&coins)
	if hives != 2 || dogs != 1 || coins != 12000000 {
		t.Fatalf("after upgrade: %d hives, %d dogs, %d milli", hives, dogs, coins)
	}
	// the new rules hold even at the database level
	if _, err := db.Exec(`INSERT INTO structures (player_id,kind,x,y) VALUES (1,'fence',6,6)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO structures (player_id,kind,x,y) VALUES (1,'lantern',6,6)`); err == nil {
		t.Fatal("two decorations on one cell must be refused by the database itself")
	}
	if _, err := db.Exec(`INSERT INTO structures (player_id,kind,x,y) VALUES (1,'hat',0,0)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO structures (player_id,kind,x,y) VALUES (1,'hat',0,0)`); err == nil {
		t.Fatal("a second hat must be refused by the database itself")
	}
}

// Phase 4: a database at schema 5 (decoration in place, a finished Pomodoro history) must come through migration 6 with every
// row intact, and the Pomodoros it already holds read as "not strict" (the mode did not exist when they were played).
func TestUpgradingFromSchema5KeepsHistoryAndDefaultsStrictToOff(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v5.db")
	raw, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	for v, name := range []string{"0001_init.sql", "0002_silo.sql", "0003_rest.sql", "0004_structures.sql", "0005_decor.sql"} {
		body, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := raw.Exec(string(body)); err != nil {
			t.Fatalf("applying %s: %v", name, err)
		}
		if _, err := raw.Exec(`INSERT INTO schema_migrations VALUES (?, '2026-10-07T12:00:00Z')`, v+1); err != nil {
			t.Fatal(err)
		}
	}
	for _, q := range []string{
		`INSERT INTO players (id,name,focus_points,lifetime_focus,coins_milli,silo_level,season,biome,created_at,last_seen_at,version)
			VALUES (1,'Luk',5,300,14258000,4,1,'spring','2026-10-01T08:00:00Z','2026-10-08T11:00:00Z',12)`,
		`INSERT INTO tags (id,player_id,name) VALUES (1,1,'tesis')`,
		`INSERT INTO pomodoros (id,player_id,plot_id,plant_type,tag_id,planned_s,started_at,paused_total_s,ended_at,reward_focus,status)
			VALUES (1,1,NULL,'apple',1,2700,'2026-10-06T08:00:00Z',60,'2026-10-06T08:46:00Z',15,'completed'),
			       (2,1,NULL,'daisy',NULL,600,'2026-10-07T08:00:00Z',0,'2026-10-07T08:10:00Z',1,'completed'),
			       (3,1,NULL,'daisy',NULL,600,'2026-10-07T09:00:00Z',0,'2026-10-07T09:05:00Z',0,'cancelled')`,
		`INSERT INTO pomodoro_events (pomodoro_id,kind,at) VALUES (1,'start','2026-10-06T08:00:00Z'),(1,'pause','2026-10-06T08:10:00Z')`,
		`INSERT INTO structures (player_id,kind,x,y) VALUES (1,'lantern',4,1),(1,'hat',0,0)`,
		`INSERT INTO settings VALUES (1,'rest_enabled','0')`,
	} {
		if _, err := raw.Exec(q); err != nil {
			t.Fatalf("setting up the v5 database: %v", err)
		}
	}
	raw.Close()

	db, err := Open(path)
	if err != nil {
		t.Fatalf("opening a schema-5 database: %v", err)
	}
	defer db.Close()
	if v, _ := SchemaVersion(db); v != 6 {
		t.Fatalf("schema version = %d, want 6", v)
	}
	var strict, count int
	if err := db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(strict), 0) FROM pomodoros`).Scan(&count, &strict); err != nil || count != 3 || strict != 0 {
		t.Fatalf("%d Pomodoros, %d strict, err %v", count, strict, err)
	}
	var planned int
	var status string
	if err := db.QueryRow(`SELECT planned_s, status FROM pomodoros WHERE id = 1`).Scan(&planned, &status); err != nil || planned != 2700 || status != "completed" {
		t.Fatalf("a Pomodoro changed: %d %s %v", planned, status, err)
	}
	var coins, events, structures, settings int64
	db.QueryRow(`SELECT coins_milli FROM players WHERE id = 1`).Scan(&coins)
	db.QueryRow(`SELECT COUNT(*) FROM pomodoro_events`).Scan(&events)
	db.QueryRow(`SELECT COUNT(*) FROM structures`).Scan(&structures)
	db.QueryRow(`SELECT COUNT(*) FROM settings`).Scan(&settings)
	if coins != 14258000 || events != 2 || structures != 2 || settings != 1 {
		t.Fatalf("rows changed: coins %d, events %d, structures %d, settings %d", coins, events, structures, settings)
	}
	// the new column accepts the flag, and rejects nothing that was valid before
	if _, err := db.Exec(`INSERT INTO pomodoros (player_id,plot_id,plant_type,planned_s,started_at,status,strict) VALUES (1,NULL,'daisy',600,'2026-10-08T08:00:00Z','running',1)`); err != nil {
		t.Fatal(err)
	}
}
