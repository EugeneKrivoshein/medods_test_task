package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/EugeneKrivoshein/medods_test_task/internal/models"
	"github.com/EugeneKrivoshein/medods_test_task/internal/postgres"
	db "github.com/EugeneKrivoshein/medods_test_task/internal/postgres"
	"github.com/EugeneKrivoshein/medods_test_task/internal/services"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	dbProvider   *db.PostgresProvider
	tokenService *services.TokenService
	repo         *postgres.TokenRepository
	log          *logrus.Logger
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func NewAuthHandler(provider *db.PostgresProvider, tokenService *services.TokenService, repo *postgres.TokenRepository) *AuthHandler {
	log := logrus.New()
	return &AuthHandler{
		dbProvider:   provider,
		tokenService: tokenService,
		repo:         repo,
		log:          log,
	}
}

func getClientIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}
	return ip
}

func (h *AuthHandler) GenerateTokens(w http.ResponseWriter, r *http.Request) {
	var req models.TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Некорректный запрос")
		return
	}
	clientIP := getClientIP(r)

	ipParts := strings.Split(clientIP, ":")
	if len(ipParts) > 1 {
		clientIP = ipParts[0]
	}

	h.log.WithFields(logrus.Fields{
		"user_id":   req.UserID,
		"client_ip": clientIP,
	}).Info("Генерация токенов")

	user, err := h.repo.FindUserByID(req.UserID)
	if err != nil {
		h.log.Errorf("Ошибка проверки пользователя: %v", err)
		h.respondWithError(w, http.StatusInternalServerError, "Не удалось проверить пользователя")
		return
	}

	if user == nil {
		defaultEmail := fmt.Sprintf("user_%s@mail.com", req.UserID)

		passwordHash, err := bcrypt.GenerateFromPassword([]byte("defaultpassword"), bcrypt.DefaultCost)
		if err != nil {
			h.log.Errorf("Ошибка хэширования пароля: %v", err)
			h.respondWithError(w, http.StatusInternalServerError, "Не удалось хэшировать пароль")
			return
		}

		newUser := &models.User{
			ID:           req.UserID,
			Email:        defaultEmail,
			PasswordHash: string(passwordHash),
		}
		if err := h.repo.CreateUser(newUser); err != nil {
			h.log.Errorf("Ошибка создания пользователя: %v", err)
			h.respondWithError(w, http.StatusInternalServerError, "Не удалось создать пользователя")
			return
		}
		h.log.Infof("Новый пользователь создан: %s", req.UserID)
	}

	tokens, err := h.tokenService.GenerateTokens(req.UserID, clientIP)
	if err != nil {
		h.log.Errorf("Ошибка генерации токенов: %v", err)
		h.respondWithError(w, http.StatusInternalServerError, "Не удалось сгенерировать токены")
		return
	}

	h.respondWithJSON(w, http.StatusOK, tokens)
}

func (h *AuthHandler) RefreshTokens(w http.ResponseWriter, r *http.Request) {
	var req models.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Errorf("Ошибка декодирования тела запроса: %v", err)
		h.respondWithError(w, http.StatusBadRequest, "Некорректный запрос")
		return
	}

	if req.RefreshToken == "" {
		h.respondWithError(w, http.StatusBadRequest, "RefreshToken не может быть пустым")
		return
	}
	clientIP := getClientIP(r)

	ipParts := strings.Split(clientIP, ":")
	if len(ipParts) > 1 {
		clientIP = ipParts[0]
	}

	h.log.WithFields(logrus.Fields{
		"refresh_token": req.RefreshToken,
		"client_ip":     clientIP,
	}).Info("Обновление токенов")

	tokens, err := h.tokenService.RefreshTokens(req.RefreshToken, clientIP)
	if err != nil {
		h.log.Errorf("Ошибка обновления токенов: %v", err)
		h.respondWithError(w, http.StatusUnauthorized, "Не удалось обновить токены")
		return
	}

	h.respondWithJSON(w, http.StatusOK, tokens)
}

func (h *AuthHandler) respondWithError(w http.ResponseWriter, code int, message string) {
	response := map[string]string{"error": message}
	h.respondWithJSON(w, code, response)
}

func (h *AuthHandler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		h.log.Errorf("Ошибка сериализации ответа: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
