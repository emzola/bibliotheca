package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"time"

	"github.com/emzola/bibliotheca/data"
)

type tokens interface {
	CreateNewToken(userID int64, ttl time.Duration, scope string) (*data.Token, error)
	DeleteAllTokensForUser(scope string, userID int64) error
}

// generateToken generates a new user token.
func generateToken(userID int64, ttl time.Duration, scope string) (*data.Token, error) {
	token := &data.Token{
		UserID: userID,
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}
	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]
	return token, nil
}

// CreateNewToken is a shortcut method which generates and creates a new token record.
func (r *repository) CreateNewToken(userID int64, ttl time.Duration, scope string) (*data.Token, error) {
	token, err := generateToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}
	err = r.createToken(token)
	return token, err
}

// createToken creates a token record.
func (r *repository) createToken(token *data.Token) error {
	query := `
		INSERT INTO tokens (hash, user_id, expiry, scope)
		VALUES ($1, $2, $3, $4)`
	args := []interface{}{token.Hash, token.UserID, token.Expiry, token.Scope}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// DeleteAllTokensForUser deletes all tokens for a specific user and scope.
func (r *repository) DeleteAllTokensForUser(scope string, userID int64) error {
	if userID < 1 {
		return ErrRecordNotFound
	}
	query := `
		DELETE FROM tokens
		WHERE scope = $1 AND user_id = $2`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := r.db.ExecContext(ctx, query, scope, userID)
	return err
}
