package models

import "time"

type Token struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	RefreshTokenHash string    `json:"refresh_token_hash"`
	ClientIP         string    `json:"client_ip"`
	CreatedAt        time.Time `json:"created_at"`
	ExpiresAt        time.Time `json:"expires_at"`
}

type TokenRequest struct {
	UserID   string `json:"user_id"`
	ClientIP string `json:"client_ip"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
	ClientIP     string `json:"client_ip"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	PasswordHash string `json:"password_hash"`
}

type EmailSender interface {
	SendEmail(to string, subject string, body string) error
}

type TokenRepository interface {
	CreateUser(User *User) error
	SaveToken(token *Token) error
	FindTokenByRefreshToken(refreshToken string) (*Token, error)
	FindUserByID(userID string) (*User, error)
}
