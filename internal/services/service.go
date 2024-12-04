package services

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/EugeneKrivoshein/medods_test_task/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type TokenService struct {
	db     models.TokenRepository
	secret string
	email  models.EmailSender
}

type MockEmailService struct{}

func NewTokenService(db models.TokenRepository, secret string, email models.EmailSender) *TokenService {
	return &TokenService{
		db:     db,
		secret: secret,
		email:  email,
	}
}

func (t *TokenService) GenerateTokens(userID, clientIP string) (*models.TokenResponse, error) {
	tokenID := uuid.New().String()

	accessTokenClaims := jwt.MapClaims{
		"user_id":   userID,
		"client_ip": clientIP,
		"exp":       time.Now().Add(15 * time.Minute).Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS512, accessTokenClaims)
	signedAccessToken, err := accessToken.SignedString([]byte(t.secret))
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации Access токена: %w", err)
	}

	refreshToken := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s|%s|%s", time.Now().String(), userID, clientIP)))

	if len(refreshToken) > 72 {
		refreshToken = refreshToken[:72]
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(refreshToken), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации хеша для Refresh токена: %w", err)
	}

	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	token := &models.Token{
		ID:               tokenID,
		UserID:           userID,
		RefreshTokenHash: string(hash),
		ClientIP:         clientIP,
		CreatedAt:        time.Now(),
		ExpiresAt:        expiresAt,
	}

	if err := t.db.SaveToken(token); err != nil {
		return nil, fmt.Errorf("ошибка сохранения Refresh токена: %w", err)
	}

	return &models.TokenResponse{
		AccessToken:  signedAccessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (t *TokenService) RefreshTokens(refreshToken, clientIP string) (*models.TokenResponse, error) {
	log := logrus.New()

	token, err := t.db.FindTokenByRefreshToken(refreshToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Warn("Refresh token не найден")
		} else {
			log.WithError(err).Error("Ошибка запроса токена из БД")
		}
		return nil, errors.New("Unauthorized")
	}

	if token.ExpiresAt.Before(time.Now()) {
		log.Warn("Refresh token истёк")
		return nil, errors.New("Unauthorized")
	}

	if token.ClientIP != clientIP {
		user, err := t.db.FindUserByID(token.UserID)
		if err == nil {
			emailErr := t.email.SendEmail(user.Email, "Security Alert", "Доступ к вашей учетной записи был получен с нового IP-адреса.")
			if emailErr != nil {
				log.WithError(emailErr).Error("Не удалось отправить email")
			} else {
				log.Infof("Предупреждение отправлено на %s", user.Email)
			}
		}
	}

	newTokens, err := t.GenerateTokens(token.UserID, clientIP)
	if err != nil {
		log.WithError(err).Error("Ошибка создания новых токенов")
		return nil, errors.New("Ошибка сервера")
	}

	log.Info("Токены успешно обновлены")
	return newTokens, nil
}

func (m *MockEmailService) SendEmail(to, subject, body string) error {
	logMessage := fmt.Sprintf("Mock Email Sent:\n  To: %s\n  Subject: %s\n  Body: %s\n", to, subject, body)
	fmt.Println(logMessage)
	return nil
}
