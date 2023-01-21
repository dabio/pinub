package main

import (
	"encoding/hex"
	"os"

	"dab.io/pinub"
	"golang.org/x/exp/slog"
)

func main() {
	secretKey, err := hex.DecodeString(env("SECRET_KEY", "7D8C9FA38B164A11843404B989E6491F"))
	if err != nil {
		slog.Error("secret key error", err)
	}

	app := &pinub.App{
		ListenAddress: env("LISTEN_ADDRESS", "127.0.0.1:8080"),
		SecretKey:     secretKey,

		DSN: env("DSN", "pinub.sqlite3"),
	}
	app.Start()
}

func env(key, defaultValue string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}

	return defaultValue
}
