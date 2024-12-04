package api

import (
	"net/http"

	"github.com/EugeneKrivoshein/medods_test_task/internal/handlers"
	"github.com/gorilla/mux"
)

func NewRouter(authHandler *handlers.AuthHandler) *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Сервер работает!"))
	}).Methods("GET")
	router.HandleFunc("/auth/tokens", authHandler.GenerateTokens).Methods("POST")
	router.HandleFunc("/auth/refresh", authHandler.RefreshTokens).Methods("POST")

	return router
}
