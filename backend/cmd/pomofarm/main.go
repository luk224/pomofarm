package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/luk224/pomofarm/backend/internal/api"
	"github.com/luk224/pomofarm/backend/internal/store"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	dbPath := env("POMOFARM_DB", "data/pomofarm.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		log.Fatal(err)
	}
	db, err := store.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	log.Fatal(api.New(db).Listen(env("POMOFARM_ADDR", ":8080")))
}
