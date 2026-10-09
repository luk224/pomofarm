package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
	_ "time/tzdata" // IANA time zones inside the binary: the container image has no tz database

	"github.com/luk224/pomofarm/backend/internal/api"
	"github.com/luk224/pomofarm/backend/internal/game"
	"github.com/luk224/pomofarm/backend/internal/service"
	"github.com/luk224/pomofarm/backend/internal/store"
)

const keepBackups = 14

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	dbPath := env("POMOFARM_DB", "data/pomofarm.db")
	backupDir := env("POMOFARM_BACKUPS", "backups")

	// `pomofarm verify <file>` checks that a backup is a healthy PomoFarm database. It never touches the live database.
	if len(os.Args) > 2 && os.Args[1] == "verify" {
		if err := store.Verify(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, "INVALID:", err)
			os.Exit(1)
		}
		fmt.Println("OK", os.Args[2])
		return
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		log.Fatal(err)
	}
	// a verified safety copy is taken before any upgrade touches an existing game
	db, err := store.OpenWithSafety(dbPath, backupDir, time.Now(), log.Printf)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// `pomofarm backup` takes one backup now and exits.
	if len(os.Args) > 1 && os.Args[1] == "backup" {
		p, err := store.Backup(db, backupDir, keepBackups, time.Now())
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(p)
		return
	}

	if err := service.New(db, game.SystemClock{}).EnsurePlayer(context.Background(), env("POMOFARM_PLAYER", "Granjero")); err != nil {
		log.Fatal(err)
	}

	stop := make(chan struct{})
	defer close(stop)
	go store.RunBackups(db, backupDir, keepBackups, 24*time.Hour, time.Hour, stop, log.Printf)
	log.Fatal(api.New(db, game.SystemClock{}).Listen(env("POMOFARM_ADDR", ":8080")))
}
