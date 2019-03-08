package postgres

import "github.com/dabio/pinub/auth"

var _ auth.Backend = &Client{}

// ByEmail searches the user table for a user with the given email address.
func (c *Client) ByEmail(email string) (*auth.User, error) {
	var u auth.User

	query := `SELECT email, password, created_at FROM users
		WHERE email = $1 LIMIT 1`
	err := c.cursor().
		QueryRow(query, email).
		Scan(&u.Email, &u.Password, &u.CreatedAt)

	return &u, err
}

// ByToken queries the database for a user by the given token.
func (c *Client) ByToken(token string) (*auth.User, error) {
	var u auth.User

	query := `SELECT email, password, token, l.active_at, u.created_at
		FROM users u JOIN logins l ON u.id = l.user_id AND l.token = $1 LIMIT 1`
	err := c.cursor().
		QueryRow(query, token).
		Scan(&u.Email, &u.Password, &u.Token, &u.ActiveAt, &u.CreatedAt)

	return &u, err
}

// NewUser creates a new entry with given parameters to users table.
func (c *Client) NewUser(email, password string) (*auth.User, error) {
	var u auth.User

	query := `INSERT INTO users (email, password) VALUES ($1, $2)
		RETURNING email, password, created_at`
	err := c.cursor().
		QueryRow(query, email, password).
		Scan(&u.Email, &u.Password, &u.CreatedAt)

	return &u, err
}

// UpdateEmail updates the email of a user.
func (c *Client) UpdateEmail(token, email string) error {
	query := `UPDATE users SET email = $1 FROM logins
		WHERE id = user_id AND token = $2`
	_, err := c.cursor().Exec(query, email, token)

	return err
}

// UpdatePassword updates the password of a user.
func (c *Client) UpdatePassword(token, passwordHash string) error {
	query := `UPDATE users SET password = $1 FROM logins
		WHERE id = user_id AND token = $2`
	_, err := c.cursor().Exec(query, passwordHash, token)

	return err
}

// NewToken creates a new token for a user identified by the given email.
func (c *Client) NewToken(email string) (string, error) {
	var token string
	query := `INSERT INTO logins (user_id) SELECT id FROM users WHERE
		email = $1 RETURNING token`
	err := c.cursor().QueryRow(query, email).Scan(&token)

	return token, err
}

// // RefreshToken updates the last seen token of the user.
// func (s *UserService) RefreshToken(ctx context.Context, token string) error {
// 	query := "UPDATE logins SET active_at = now() WHERE token = $1"
// 	_, err := s.client.ExecContext(ctx, query, token)

// 	return err
// }
