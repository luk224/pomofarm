package main

import (
	"log"
	"os"

	"github.com/luk224/pomofarm/backend/internal/api"
)

func main() {
	addr := os.Getenv("POMOFARM_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Fatal(api.New().Listen(addr))
}
