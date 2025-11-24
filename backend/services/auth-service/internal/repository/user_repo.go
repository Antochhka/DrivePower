package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"drivepower/backend/services/auth-service/internal/models"
)

// ErrUserNotFound represents missing user rows.
var ErrUserNotFound = errors.New("user not found")

// UserRepository handles CRUD for users table.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository returns repository instance.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user.
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	user.Email = strings.ToLower(strings.TrimSpace(user.Email))
	const query = `
		INSERT INTO users (email, password_hash, role)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	return r.db.QueryRowContext(ctx, query, user.Email, user.PasswordHash, user.Role).
		Scan(&user.ID, &user.CreatedAt)
}

// GetByEmail fetches a user by email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	const query = `
		SELECT id, email, password_hash, role, created_at
		FROM users
		WHERE email = $1
		LIMIT 1
	`
	row := r.db.QueryRowContext(ctx, query, strings.ToLower(strings.TrimSpace(email)))
	var user models.User
	if err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

