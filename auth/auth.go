package auth

import (
	"errors"
	"regexp"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrMismatchedHashAndPassword is the error returned from
	// CompareHashAndPassword when a password and hash do not match.
	ErrMismatchedHashAndPassword = errors.New("Password is not correct")
	// ErrUnknownEmail is the error returned when the email address is cannot
	// be found the backend.
	ErrUnknownEmail = errors.New("Email address is not known")
	// ErrInvalidEmail is an error returned when the given email address is
	// not in a valid format. @see emailRe.
	ErrInvalidEmail = errors.New("Email is not in a valid format")
	// ErrEmailExists will be returned when the given email is already stored.
	ErrEmailExists = errors.New("The email address is already in use by another account")
	// ErrPasswordTooShort will be returned when the given password is too short.
	ErrPasswordTooShort = errors.New("Password is too short")

	emailRe = regexp.MustCompile(`.+@.+\..+`)
)

// User has all attributes of a user.
type User struct {
	// ID        string
	Email     string
	Password  string
	Token     string
	CreatedAt *time.Time
	ActiveAt  *time.Time
}

// Backend will be used to fetch data from.
type Backend interface {
	ByEmail(string) (*User, error)
	ByToken(string) (*User, error)

	NewUser(string, string) (*User, error)
	NewToken(string) (string, error)

	UpdateEmail(string, string) error
	UpdatePassword(string, string) error
}

// Config has all the required settings.
type Config struct {
	MinPassLen int
}

// Client hold all relevant information to retrieve a user from a backend.
type Client struct {
	backend Backend
	config  Config
}

// NewClient creates a new Auth object with necessary information to get user
// from a backend.
func NewClient(config Config, backend Backend) *Client {
	auth := Client{
		backend: backend,
		config:  config,
	}

	return &auth
}

// VerifyPassword will sign in a user with an email and password.
func (a *Client) VerifyPassword(email, passwd string) (*User, error) {
	u, err := a.backend.ByEmail(email)
	if err != nil {
		return nil, ErrUnknownEmail
	}

	if err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(passwd)); err != nil {
		return nil, ErrMismatchedHashAndPassword
	}

	if u.Token, err = a.backend.NewToken(email); err != nil {
		return nil, err
	}

	return u, nil
}

// SignupNewUser will create a new user by email and password.
func (a *Client) SignupNewUser(email, password string) (*User, error) {
	// check for valid email address
	if !emailRe.MatchString(email) {
		return nil, ErrInvalidEmail
	}
	// check for valid password length
	if len(password) < a.config.MinPassLen {
		return nil, ErrPasswordTooShort
	}

	// check if email is already in use
	if u, _ := a.backend.ByEmail(email); u.Email != "" {
		return nil, ErrEmailExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	// create a new user
	u, err := a.backend.NewUser(email, string(hash))
	if err != nil {
		return nil, err
	}
	if u.Token, err = a.backend.NewToken(u.Email); err != nil {
		return nil, err
	}

	return u, nil
}

// AccountInfo returns a user object with all stored account information
// available.
func (a *Client) AccountInfo(token string) (*User, error) {
	return a.backend.ByToken(token)
}

// ChangeEmail changes a users email address to the given one. Use token to
// identify a user.
func (a *Client) ChangeEmail(token string, email string) error {
	// check for valid email address
	if !emailRe.MatchString(email) {
		return ErrInvalidEmail
	}
	// check if email is already in use
	if u, _ := a.backend.ByEmail(email); u.Email != "" {
		return ErrEmailExists
	}

	return a.backend.UpdateEmail(token, email)
}

// ChangePassword changes a users password to the given one. Use token to
// identify a user.
func (a *Client) ChangePassword(token, password string) error {
	if len(password) < a.config.MinPassLen {
		return ErrPasswordTooShort
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return a.backend.UpdatePassword(token, string(hash))
}
