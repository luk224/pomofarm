package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

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
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		log.Fatal(err)
	}
	db, err := store.Open(dbPath)
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
