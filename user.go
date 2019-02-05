package main

import "time"

type user struct {
	ID        string
	Email     string
	Password  string
	Token     string
	CreatedAt *time.Time
	ActiveAt  *time.Time
}
