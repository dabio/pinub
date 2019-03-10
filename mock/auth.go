package mock

import "github.com/dabio/pinub/auth"

var _ auth.Backend = &Backend{}

type Backend struct {
	ByEmailFn      func(string) (*auth.User, error)
	ByEmailInvoked bool

	ByTokenFn      func(string) (*auth.User, error)
	ByTokenInvoked bool

	NewUserFn      func(string, string) (*auth.User, error)
	NewUserInvoked bool

	NewTokenFn      func(string) (string, error)
	NewTokenInvoked bool

	UpdateEmailFn      func(string, string) error
	UpdateEmailInvoked bool

	UpdatePasswordFn      func(string, string) error
	UpdatePasswordInvoked bool
}

// ByEmail helper
func (b *Backend) ByEmail(email string) (*auth.User, error) {
	b.ByEmailInvoked = true
	return b.ByEmailFn(email)
}

// ByToken helper
func (b *Backend) ByToken(token string) (*auth.User, error) {
	b.ByTokenInvoked = true
	return b.ByTokenFn(token)
}

// NewUser helper
func (b *Backend) NewUser(email, password string) (*auth.User, error) {
	b.NewUserInvoked = true
	return b.NewUserFn(email, password)
}

// NewToken helper
func (b *Backend) NewToken(email string) (string, error) {
	b.NewTokenInvoked = true
	return b.NewTokenFn(email)
}

// UpdateEmail helper
func (b *Backend) UpdateEmail(token, email string) error {
	b.UpdateEmailInvoked = true
	return b.UpdateEmailFn(token, email)
}

// UpdatePassword helper
func (b *Backend) UpdatePassword(token, password string) error {
	b.UpdatePasswordInvoked = true
	return b.UpdatePasswordFn(token, password)
}
