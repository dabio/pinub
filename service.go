package main

import "context"

type userService interface {
	UserByEmail(context.Context, string) (*user, error)
	UserByToken(context.Context, string) (*user, error)

	CreateUser(context.Context, *user) error
	UpdateUser(context.Context, *user) error
	DeleteUser(context.Context, *user) error

	UserAddToken(context.Context, *user) error
	UserRefreshToken(context.Context, *user) error
}

type linkService interface {
	// AllLinks(context.Context, *user) ([]link, error)
	CreateLink(context.Context, *link, *user) error
	DeleteLink(context.Context, *link, *user) error
}

type client interface {
	UserService() userService
	LinkService() linkService
}
