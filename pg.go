package main

import (
	"context"
	"database/sql"
	"log"
	"sync"

	_ "github.com/lib/pq"
)

// Ensure postgres implements interfaces
var (
	_ client      = &pg{}
	_ userService = &pg{}
	_ linkService = &pg{}
)

type pg struct {
	*sql.DB

	init       sync.Once
	dataSource string
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
}

func (c *pg) LinkService() linkService {
	c.init.Do(func() {
		c.initPg()
	})
	return c
}

func (c *pg) UserService() userService {
	c.init.Do(func() {
		c.initPg()
	})
	return c
}

// UserByEmail searches the user table for a user with the given email address.
func (c *pg) UserByEmail(ctx context.Context, email string) (*user, error) {
	var u user

	query := "SELECT id, email, password FROM users WHERE email = $1 LIMIT 1"
	err := c.
		QueryRowContext(ctx, query).
		Scan(&u.ID, &u.Email, &u.Password)

	return &u, err
}

// UserByToken queries the database for a user by the given token.
func (c *pg) UserByToken(ctx context.Context, token string) (*user, error) {
	var u user

	query := `SELECT id, email, password, token, active_at FROM users u
		JOIN logins l ON u.id = l.user_id AND l.token = $1 LIMIT 1`
	err := c.
		QueryRowContext(ctx, query).
		Scan(&u.ID, &u.Email, &u.Password, &u.Token, &u.ActiveAt)

	return &u, err
}

// CreateUser stores the given user object on the users table.
func (c *pg) CreateUser(ctx context.Context, u *user) error {
	query := "INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id"
	err := c.QueryRowContext(ctx, query, u.Email, u.Password).Scan(&u.ID)

	return err
}

// UpdateUserEmail updates the email of a user.
func (c *pg) UpdateUserEmail(ctx context.Context, u *user) error {
	query := "UPDATE users SET email = $1 WHERE id = $2"
	_, err := c.ExecContext(ctx, query, u.Email, u.ID)

	return err
}

// UpdateUserPassword updates the password of a user.
func (c *pg) UpdateUserPassword(ctx context.Context, u *user) error {
	query := "UPDATE users SET password = $1 WHERE id = $2"
	_, err := c.ExecContext(ctx, query, u.Password, u.ID)

	return err
}

// DeleteUser removes the given user object from the users table.
// func (c *pg) DeleteUser(ctx context.Context, u *user) error {
// 	query := "DELETE FROM users WHERE id = $1"
// 	_, err := c.ExecContext(ctx, query, u.ID)

// 	return err
// }

// AddToken creates a new token for the given user object and sets the last
// active date to now.
func (c *pg) UserAddToken(ctx context.Context, u *user) error {
	query := "INSERT INTO logins (user_id) VALUES ($1) RETURNING token, active_at"
	err := c.QueryRowContext(ctx, query, u.ID).Scan(&u.Token, &u.ActiveAt)

	return err
}

// RefreshToken updates the last seen token of the user.
func (c *pg) UserRefreshToken(ctx context.Context, u *user) error {
	query := "UPDATE logins SET active_at = now() WHERE token = $1 RETURNING active_at"
	err := c.QueryRowContext(ctx, query, u.Token).Scan(&u.ActiveAt)

	return err
}

func (c *pg) link(ctx context.Context, l *link) error {
	query := "SELECT id FROM links WHERE url = $1 LIMIT 1"
	if err := c.QueryRowContext(ctx, query, l.URL).Scan(&l.ID); err != nil {
		return err
	}

	return nil
}

func (c *pg) createLink(ctx context.Context, l *link) error {
	query := "INSERT INTO links (url) VALUES ($1) RETURNING id"
	if err := c.QueryRowContext(ctx, query, l.URL).Scan(&l.ID); err != nil {
		return err
	}

	return nil
}

func (c *pg) linkForUser(ctx context.Context, l *link, u *user) error {
	query := `SELECT created_at FROM user_links
		WHERE link_id = $1 AND user_id = $2 LIMIT 1`
	if err := c.QueryRowContext(ctx, query, l.ID, u.ID).Scan(&l.CreatedAt); err != nil {
		return err
	}
	return nil
}

// CreateLink creates a new link for the given user. Four steps are necessary:
//   1. check if link exists in table
//   2. if no - create it
//   3. check if user has a relation to link
//   4. if no - create it
// We can make a shortcut when link does not exists, we create it and also
// create the relation to the user.
func (c *pg) CreateLinkForUser(ctx context.Context, l *link, u *user) error {
	if err := c.link(ctx, l); err == sql.ErrNoRows {
		if err = c.createLink(ctx, l); err != nil {
			return err
		}
	}

	if err := c.linkForUser(ctx, l, u); err == sql.ErrNoRows {
		query := "INSERT INTO user_links (link_id, user_id) VALUES ($1, $2) RETURNING created_at"
		if err = c.QueryRowContext(ctx, query, l.ID, u.ID).Scan(&l.CreatedAt); err != nil {
			return err
		}
	}

	return nil
}

// DeleteLink remove the link for the given user. Also removes the link
// when no user stored that link anymore.
func (c *pg) DeleteLinkForUser(ctx context.Context, l *link, u *user) error {
	query := "DELETE FROM user_links WHERE user_id = $1 AND link_id = $2"
	if _, err := c.ExecContext(ctx, query, u.ID, l.ID); err != nil {
		return err
	}

	query = `DELETE FROM links WHERE id = $1 AND
		 (SELECT count(link_id) FROM user_links WHERE link_id = $1) = 0`
	_, err := c.ExecContext(ctx, query, l.ID)

	return err
}
