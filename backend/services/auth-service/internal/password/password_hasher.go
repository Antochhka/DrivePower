package password

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// Hasher defines password hashing contract.
type Hasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

// BcryptHasher implements Hasher using bcrypt.
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher returns a bcrypt-backed password hasher.
func NewBcryptHasher(cost int) *BcryptHasher {
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	return &BcryptHasher{cost: cost}
}

// Hash converts plain password into hash.
func (h *BcryptHasher) Hash(password string) (string, error) {
	if password == "" {
		return "", errors.New("password: empty password")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Compare checks if provided password matches stored hash.
func (h *BcryptHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

