package pinub

import (
	"context"
	"database/sql"
	"net/mail"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        string
	Email     string
	Password  string
	CreatedAt *time.Time

	Token string
}

type Login struct {
	UserID    string
	Token     string
	ActiveAt  *time.Time
	CreatedAt *time.Time
}

type UserService struct {
	DB *sql.DB
}

func (service *UserService) LoginUser(ctx context.Context, email, password string) (*User, error) {
	// try to parse for a valid email address
	parsedEmail, err := mail.ParseAddress(email)
	if err != nil {
		return nil, err
	}

	// get the user from database
	user, err := service.UserByEmail(ctx, parsedEmail.Address)
	if err != nil {
		return nil, err
	}

	// compare password with database password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, err
	}

	// create token
	login, err := service.CreateLoginForUser(ctx, user)
	if err != nil {
		return nil, err
	}
	user.Token = login.Token

	return user, nil
}

func (service *UserService) UserByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}

	query := "SELECT id, email, password, created_at FROM users WHERE email = $1 LIMIT 1"
	err := service.DB.QueryRowContext(ctx, query, email).
		Scan(&user.ID, &user.Email, &user.Password, &user.CreatedAt)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (service *UserService) CreateLoginForUser(ctx context.Context, user *User) (*Login, error) {
	login := &Login{}

	query := "INSERT INTO logins (user_id, token) VALUES ($1, $2) RETURNING user_id, token, active_at"
	err := service.DB.QueryRowContext(ctx, query, user.ID, uuid.NewString()).
		Scan(&login.UserID, &login.Token, &login.ActiveAt)
	if err != nil {
		return nil, err
	}

	return login, err
}
