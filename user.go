package main

import (
	"fmt"
	"regexp"
	"time"
)

const minPassLen = 4

var emailRe = regexp.MustCompile(`.+@.+\..+`)

type user struct {
	ID        string
	Email     string
	Password  string
	Token     string
	CreatedAt *time.Time
	ActiveAt  *time.Time
}

func (u *user) Validate() error {
	if !emailRe.MatchString(u.Email) {
		return fmt.Errorf("%v is not a valid email address", u.Email)
	}

	if len(u.Password) < minPassLen {
		return fmt.Errorf("password is too short")
	}

	return nil
}
