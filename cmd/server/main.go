package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/dabio/pinub"
	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", env("DATABASE_URL", "db/db.sqlite3"))
	if err != nil {
		log.Fatalf("could not open database: %v", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatalf("could not open database: %v", err)
	}

	server := pinub.App{
		DB:         db,
		Address:    env("ADDRESS", ":8080"),
		StaticBase: env("STATIC_BASE", "/static"),
	}
	server.Start()
}

func env(key, defaultValue string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}

	return defaultValue
}
