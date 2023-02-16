package pinub

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        int
	Email     string
	Password  string
	Token     string
	CreatedAt *time.Time
}

type Link struct {
	ID        int
	URL       string
	CreatedAt *time.Time
}

type UserService struct {
	DB *sql.DB
}

func (us *UserService) ByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}

	query := "SELECT id, email, password, created_at FROM users WHERE email = $1 LIMIT 1;"
	err := us.DB.
		QueryRowContext(ctx, query, email).
		Scan(&user.ID, &user.Email, &user.Password, &user.CreatedAt)

	return user, err
}

func (us *UserService) ByToken(ctx context.Context, token string) (*User, error) {
	user := &User{}

	query := "SELECT u.id, u.email, u.password, u.created_at, l.token FROM users u " +
		" JOIN logins l ON u.id = l.user_id AND l.token = $1 LIMIT 1;"
	err := us.DB.
		QueryRowContext(ctx, query, token).
		Scan(&user.ID, &user.Email, &user.Password, &user.CreatedAt, &user.Token)

	return user, err
}

func (us *UserService) CreateUser(ctx context.Context, user *User) error {
	query := "INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id, created_at;"

	return us.DB.
		QueryRowContext(ctx, query, user.Email, user.Password).
		Scan(&user.ID, &user.CreatedAt)
}

func (us *UserService) CreateToken(ctx context.Context, user *User) error {
	uuid := uuid.NewString()
	query := "INSERT INTO logins (user_id, token) VALUES ($1, $2) RETURNING token;"

	return us.DB.
		QueryRowContext(ctx, query, user.ID, uuid).
		Scan(&user.Token)
}

func (us *UserService) UpdateToken(ctx context.Context, token string) error {
	query := "UPDATE logins SET active_at = datetime('now') WHERE token = $1"
	_, err := us.DB.ExecContext(ctx, query, token)

	return err
}

func (us *UserService) UpdateEmail(ctx context.Context, user *User, email string) error {
	query := "UPDATE users SET email = $1 WHERE id = $2 RETURNING email;"

	return us.DB.
		QueryRowContext(ctx, query, email, user.ID).
		Scan(&user.Email)
}

func (us *UserService) UpdatePassword(ctx context.Context, user *User, password string) error {
	query := "UPDATE users SET password = $1 WHERE id = $2 RETURNING password;"

	return us.DB.
		QueryRowContext(ctx, query, password, user.ID).
		Scan(&user.Email)
}

func (us *UserService) Links(ctx context.Context, user *User) ([]Link, error) {
	query := `
		SELECT l.id, l.url, ul.created_at FROM links l
		JOIN user_links ul ON l.id = ul.link_id AND ul.user_id = $1
		ORDER BY ul.created_at DESC;`

	rows, err := us.DB.QueryContext(ctx, query, user.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []Link
	for rows.Next() {
		var link Link
		if err := rows.Scan(&link.ID, &link.URL, &link.CreatedAt); err != nil {
			return nil, err
		}

		links = append(links, link)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return links, err
}

func (us *UserService) Addlink(ctx context.Context, user *User, link *Link) error {
	// check for existing entry
	query := "SELECT id FROM links WHERE url = $1;"
	err := us.DB.QueryRowContext(ctx, query, link.URL).Scan(&link.ID)
	// create new entry
	if err == sql.ErrNoRows {
		query = "INSERT INTO links (url) VALUES ($1) RETURNING ID;"
		err = us.DB.QueryRowContext(ctx, query, link.URL).Scan(&link.ID)
	}
	if err != nil {
		return err
	}

	// check for existing link
	query = "SELECT created_at FROM user_links WHERE user_id = $1 AND link_id = $2;"
	err = us.DB.QueryRowContext(ctx, query, user.ID, link.ID).Scan(&link.CreatedAt)
	if err == sql.ErrNoRows {
		query = "INSERT INTO user_links (user_id, link_id) VALUES ($1, $2) RETURNING created_at;"
		err = us.DB.QueryRowContext(ctx, query, user.ID, link.ID).Scan(&link.CreatedAt)
	}

	// TODO: figure out an elegant way to run this only when entry was not created
	query = "UPDATE user_links SET created_at = $1 " +
		" WHERE user_id = $2 AND link_id = $3 RETURNING created_at;"

	return us.DB.
		QueryRowContext(ctx, query, time.Now().UTC().Format("2006-01-02 15:04:05"), user.ID, link.ID).
		Scan(&link.CreatedAt)
}
