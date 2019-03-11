package auth_test

import (
	"errors"
	"testing"

	"github.com/dabio/pinub/auth"
	"github.com/dabio/pinub/mock"
	"golang.org/x/crypto/bcrypt"
)

const uuid = "280136e0-39ae-4bed-923e-2c04f36a3570"

func newClient(mock *mock.Backend) *auth.Client {
	config := auth.Config{MinPassLen: 4}

	c := auth.NewClient(config, mock)

	return c
}

func validByEmail(email, password string) func(string) (*auth.User, error) {
	return func(string) (*auth.User, error) {
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		u := auth.User{
			Email:    email,
			Password: string(hash),
		}
		return &u, nil
	}
}

func validByToken(email, password string) func(string) (*auth.User, error) {
	return func(token string) (*auth.User, error) {
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		u := auth.User{
			Email:    email,
			Password: string(hash),
			Token:    token,
		}
		return &u, nil
	}
}

func validNewUser(email, hash string) func(string, string) (*auth.User, error) {
	return func(string, string) (*auth.User, error) {
		u := auth.User{
			Email:    email,
			Password: hash,
		}
		return &u, nil
	}
}

func TestVerifyPassword(t *testing.T) {
	email := "test@email.com"

	m := mock.Backend{}
	m.ByEmailFn = validByEmail(email, "pass")
	m.NewTokenFn = func(string) (string, error) {
		return uuid, nil
	}

	c := newClient(&m)
	u, _ := c.VerifyPassword(email, "pass")
	if u.Email != email {
		t.Errorf("Email is not equal. wants %v, have %v", email, u.Email)
	}
	if u.Token != uuid {
		t.Errorf("Token is not valid. wants %v, have %v", uuid, u.Token)
	}

	if _, err := c.VerifyPassword(email, "pass1"); err != auth.ErrMismatchedHashAndPassword {
		t.Error("Password should be wrong")
	}
}

func TestSignupNewUser(t *testing.T) {
	m := mock.Backend{}
	c := newClient(&m)

	if _, err := c.SignupNewUser("blah", ""); err != auth.ErrInvalidEmail {
		t.Error("Email should be invalid")
	}
	if _, err := c.SignupNewUser("test@email.com", "123"); err != auth.ErrPasswordTooShort {
		t.Error("Password should be too short")
	}

	const email = "test@email.com"
	m.ByEmailFn = validByEmail(email, "password")

	if _, err := c.SignupNewUser("test@email.com", "pass"); err != auth.ErrEmailExists {
		t.Error("Email should exist in database.")
	}

	m.NewUserFn = validNewUser(email, "password")
	m.ByEmailFn = func(string) (*auth.User, error) {
		var u auth.User
		return &u, errors.New("That user does not exist.")
	}
	m.NewTokenFn = func(string) (string, error) {
		return uuid, nil
	}

	if _, err := c.SignupNewUser(email, "password"); err != nil {
		t.Errorf("Valid user should be returned %v", err)
	}
}

func TestAccountInfo(t *testing.T) {
	m := mock.Backend{}
	c := newClient(&m)

	const email = "test@email.com"
	m.ByTokenFn = validByToken(email, "password")
	u, err := c.AccountInfo(uuid)
	if err != nil {
		t.Error("No error should occour")
	}
	if u.Token != uuid {
		t.Errorf("Token should be %v", uuid)
	}
	if u.Email != email {
		t.Errorf("Email should be %v", email)
	}
}

func TestChangeEmail(t *testing.T) {
	m := mock.Backend{}
	c := newClient(&m)

	if err := c.ChangeEmail(uuid, "email"); err == nil {
		t.Error("Email is invalid.")
	}

	const email = "test@email.com"

	m.ByEmailFn = validByEmail(email, "password")
	if err := c.ChangeEmail(uuid, email); err != auth.ErrEmailExists {
		t.Error("Email should exist already.")
	}

	m.ByEmailFn = func(string) (*auth.User, error) {
		var u auth.User
		return &u, errors.New("a random error")
	}
	m.UpdateEmailFn = func(string, string) error {
		return nil
	}
	if err := c.ChangeEmail(uuid, email); err != nil {
		t.Error("Change email should be valid.")
	}
}

func TestChangePassword(t *testing.T) {
	m := mock.Backend{}
	c := newClient(&m)

	if err := c.ChangePassword(uuid, "123"); err != auth.ErrPasswordTooShort {
		t.Error("Password should be too short.")
	}

	m.UpdatePasswordFn = func(string, string) error {
		return nil
	}
	if err := c.ChangePassword(uuid, "password"); err != nil {
		t.Error("Password change should work.")
	}
}
