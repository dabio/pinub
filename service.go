package main

type userService interface {
	// ByEmail(string) (*user, error)
	// ByToken(string) (*user, error)

	// Create(*user) error
	// Update(*user) error
	// Delete(*user) error

	// AddToken(*user) error
	// RefreshToken(*user) error
}

type linkService interface {
	// All(*user) ([]link, error)
	// Create(*link, *user) error
	// Delete(*link, *user) error
}

type client interface {
	UserService() userService
	LinkService() linkService
}
