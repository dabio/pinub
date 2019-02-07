package main

import (
	"context"
	"database/sql"
	"log"
	"sync"

	_ "github.com/lib/pq"
)

// Ensure pg and pgService implements interfaces
var (
	_ client      = &pg{}
	_ userService = &pgService{}
	_ linkService = &pgService{}
)

type pg struct {
	*sql.DB

	init       sync.Once
	dataSource string

	service pgService
}

func newPg(dataSource string) *pg {
	c := &pg{dataSource: dataSource}

	return c
}

// it would be wise to use this method with sync.Once
func (c *pg) initPg() {
	log.Println("initPg")
	db, err := sql.Open("postgres", c.dataSource)
	if err != nil {
		panic("cannot connect to database")
	}
	if err = db.Ping(); err != nil {
		panic("cannot ping to database")
	}

	c.DB = db
	c.service.Client = c
}

func (c *pg) LinkService() linkService {
	c.init.Do(func() {
		c.initPg()
	})
	return &c.service
}

func (c *pg) UserService() userService {
	c.init.Do(func() {
		c.initPg()
	})
	return &c.service
}

type pgService struct {
	Client *pg
}

// UserByEmail searches the user table for a user with the given email address.
func (s *pgService) UserByEmail(ctx context.Context, email string) (*user, error) {
	var u user

	query := "SELECT id, email, password FROM users WHERE email = $1 LIMIT 1"
	err := s.Client.
		QueryRowContext(ctx, query).
		Scan(&u.ID, &u.Email, &u.Password)

	return &u, err
}

// UserByToken queries the database for a user by the given token.
func (s *pgService) UserByToken(ctx context.Context, token string) (*user, error) {
	var u user

	query := `SELECT id, email, password, token, active_at FROM users u
		JOIN logins l ON u.id = l.user_id AND l.token = $1 LIMIT 1`
	err := s.Client.
		QueryRowContext(ctx, query).
		Scan(&u.ID, &u.Email, &u.Password, &u.Token, &u.ActiveAt)

	return &u, err
}

// CreateUser stores the given user object on the users table. Does not
// create a new user when the given users email address is already in the
// database.
func (s *pgService) CreateUser(ctx context.Context, u *user) error {
	query := "SELECT id FROM users WHERE email = $1 LIMIT 1"
	if err := s.Client.QueryRowContext(ctx, query, u.Email).Scan(&u.ID); err == nil {
		return nil
	}

	query = "INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id"
	err := s.Client.QueryRowContext(ctx, query, u.Email, u.Password).Scan(&u.ID)

	return err
}

// UpdateUser updates the fields of the user object.
func (s *pgService) UpdateUser(ctx context.Context, u *user) error {
	query := "UPDATE users SET (email, password) = ($1, $2) WHERE id = $3"
	_, err := s.Client.ExecContext(ctx, query, u.Email, u.Password, u.ID)

	return err
}

// DeleteUser removes the given user object from the users table.
func (s *pgService) DeleteUser(ctx context.Context, u *user) error {
	query := "DELETE FROM users WHERE id = $1"
	_, err := s.Client.ExecContext(ctx, query, u.ID)

	return err
}

// AddToken creates a new token for the given user object and sets the last
// active date to now.
func (s *pgService) UserAddToken(ctx context.Context, u *user) error {
	query := "INSERT INTO logins (user_id) VALUES ($1) RETURNING token, active_at"
	err := s.Client.QueryRowContext(ctx, query, u.ID).Scan(&u.Token, &u.ActiveAt)

	return err
}

// RefreshToken updates the last seen token of the user.
func (s *pgService) UserRefreshToken(ctx context.Context, u *user) error {
	query := "UPDATE logins SET active_at = now() WHERE token = $1 RETURNING active_at"
	err := s.Client.QueryRowContext(ctx, query, u.Token).Scan(&u.ActiveAt)

	return err
}

// CreateLink creates a new link for the given user. Four steps are necessary:
//   1. check if link exists in table
//   2. if no - create it
//   3. check if user has a relation to link
//   4. if no - create it
// We can make a shortcut when link does not exists, we create it and also
// create the relation to the user.
func (s *pgService) CreateLink(ctx context.Context, l *link, u *user) error {
	var err error
	var query string

	query = "SELECT id FROM links WHERE url = $1 LIMIT 1"
	if err = s.Client.QueryRowContext(ctx, query, l.URL).Scan(&l.ID); err == sql.ErrNoRows {
		query = "INSERT INTO links (url) VALUES ($1) RETURNING id"
		if err = s.Client.QueryRowContext(ctx, query, l.URL).Scan(&l.ID); err != nil {
			return err
		}
	}

	query = "SELECT created_at FROM user_links WHERE link_id = $1 AND user_id = $2 LIMIT 1"
	if err = s.Client.QueryRowContext(ctx, query, l.ID, u.ID).Scan(&l.CreatedAt); err == sql.ErrNoRows {
		query = "INSERT INTO user_links (link_id, user_id) VALUES ($1, $2) RETURNING created_at"
		if err = s.Client.QueryRowContext(ctx, query, l.ID, u.ID).Scan(&l.CreatedAt); err != nil {
			return err
		}
	}

	return nil
}

// DeleteLink remove the link for the given user. Also removes the link
// when no user stored that link anymore.
func (s *pgService) DeleteLink(ctx context.Context, l *link, u *user) error {
	query := "DELETE FROM user_links WHERE user_id = $1 AND link_id = $2"
	if _, err := s.Client.ExecContext(ctx, query, u.ID, l.ID); err != nil {
		return err
	}

	query = `DELETE FROM links WHERE id = $1 AND
		 (SELECT count(link_id) FROM user_links WHERE link_id = $1) = 0`
	_, err := s.Client.ExecContext(ctx, query, l.ID)

	return err
}
