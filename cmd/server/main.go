package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/dabio/pinub"
	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", env("DATABASE_URL", "db/development.sqlite3"))
	if err != nil {
		log.Fatal("could not open database")
	}
	defer db.Close()

	server := pinub.App{
		DB:         db,
		Port:       ":" + env("PORT", "8080"),
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
