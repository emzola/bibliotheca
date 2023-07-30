package data

import (
	"errors"
	"time"

	"github.com/emzola/bibliotheca/internal/validator"
	"golang.org/x/crypto/bcrypt"
)

var AnonymousUser = &User{}

// Check if a user instance is the anonymous user.
func (u *User) IsAnonymous() bool {
	return u == AnonymousUser
}

// User defines a user model.
type User struct {
	ID            int64     `json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	Name          string    `json:"name"`
	Email         string    `json:"email"`
	Password      password  `json:"-"`
	Activated     bool      `json:"activated"`
	DownloadCount int8      `json:"-"`
	Version       int32     `json:"-"`
}

// password defines the plaintext and hashed versions of a user's password.
// The plaintext field is a *pointer* to a string, so that we're able
// to distinguish between a plaintext password not being present in the struct at
// all, versus a plaintext password which is the empty string.
type password struct {
	Plaintext *string
	Hash      []byte
}

// Set calculates the bcrypt hash of a plaintext password.
func (p *password) Set(plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	if err != nil {
		return err
	}
	p.Plaintext = &plaintextPassword
	p.Hash = hash
	return nil
}

// Matches checks whether the provided plaintext password matches the hashed
// password stored in the User model, returning true if it matches and false otherwise.
func (p *password) Matches(plaintextPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.Hash, []byte(plaintextPassword))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}
	return true, nil
}

func ValidateName(v *validator.Validator, name string) {
	v.Check(name != "", "name", "must be provided")
	v.Check(len(name) <= 500, "name", "must not be more than 500 bytes long")
}
func ValidateEmail(v *validator.Validator, email string) {
	v.Check(email != "", "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be a valid email address")
}

func ValidatePasswordPlaintext(v *validator.Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(password) <= 72, "password", "must not be more than 72 bytes long")
}

func ValidateUser(v *validator.Validator, user *User) {
	v.Check(user.Name != "", "name", "must be provided")
	v.Check(len(user.Name) <= 500, "name", "must not be more than 500 bytes long")
	ValidateEmail(v, user.Email)
	if user.Password.Plaintext != nil {
		ValidatePasswordPlaintext(v, *user.Password.Plaintext)
	}
	if user.Password.Hash == nil {
		panic("missing password hash for user")
	}
}
