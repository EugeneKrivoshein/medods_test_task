package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/EugeneKrivoshein/medods_test_task/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type TokenRepository struct {
	Provider *PostgresProvider
}

func (r *TokenRepository) CreateUser(user *models.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, created_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.Provider.db.Exec(query, user.ID, user.Email, user.PasswordHash, time.Now())
	return err
}

func (r *TokenRepository) SaveToken(token *models.Token) error {
	_, err := r.Provider.db.Exec(`
		INSERT INTO tokens (id, user_id, refresh_token_hash, client_ip, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		token.ID, token.UserID, token.RefreshTokenHash, token.ClientIP, token.CreatedAt, token.ExpiresAt)
	return err
}

func (r *TokenRepository) FindTokenByRefreshToken(refreshToken string) (*models.Token, error) {
	rows, err := r.Provider.db.Query(`
		SELECT id, user_id, refresh_token_hash, client_ip, created_at, expires_at
		FROM tokens
	`)
	if err != nil {
		return nil, fmt.Errorf("не удалось выполнить запрос: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var token models.Token
		var refreshTokenHash string

		err := rows.Scan(&token.ID, &token.UserID, &refreshTokenHash, &token.ClientIP, &token.CreatedAt, &token.ExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("не удалось прочитать строку: %w", err)
		}

		if bcrypt.CompareHashAndPassword([]byte(refreshTokenHash), []byte(refreshToken)) == nil {
			token.RefreshTokenHash = refreshTokenHash
			return &token, nil
		}
	}

	return nil, sql.ErrNoRows
}

func (r *TokenRepository) FindUserByID(userID string) (*models.User, error) {
	var user models.User
	var id, email string

	err := r.Provider.db.QueryRow(`
		SELECT id, email FROM users WHERE id = $1
		`, userID).Scan(&id, &email)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("не удалось выполнить запрос: %w", err)
	}

	user = models.User{
		ID:    id,
		Email: email,
	}

	return &user, nil
}
