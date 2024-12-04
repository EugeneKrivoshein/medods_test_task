package main

import (
	"net/http"

	"github.com/EugeneKrivoshein/medods_test_task/config"
	"github.com/EugeneKrivoshein/medods_test_task/internal/api"
	"github.com/EugeneKrivoshein/medods_test_task/internal/handlers"
	"github.com/EugeneKrivoshein/medods_test_task/internal/postgres"
	"github.com/EugeneKrivoshein/medods_test_task/internal/postgres/migrations"
	"github.com/EugeneKrivoshein/medods_test_task/internal/services"
	"github.com/sirupsen/logrus"
)

func main() {
	log := logrus.New()

	log.Info("Запуск приложения")
	log.Debug("Загрузка конфигурации")

	cfg, err := config.LoadConfig("config.env")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}
	log.Infof("Конфигурация загружена: %+v", cfg)

	log.Debug("Подключение к базе данных")
	connect, err := postgres.NewPostgresProvider()
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer func() {
		log.Debug("Закрытие подключения к базе данных")
		connect.Close()
	}()

	if err := migrations.RunMigrations(connect, cfg.MigrationPath); err != nil {
		log.Fatalf("Ошибка выполнения миграций: %v", err)
	}

	tokenService := services.NewTokenService(&postgres.TokenRepository{Provider: connect}, cfg.JWTSecret, &services.MockEmailService{})
	authHandler := handlers.NewAuthHandler(connect, tokenService, &postgres.TokenRepository{Provider: connect})

	router := api.NewRouter(authHandler)

	log.Infof("Сервер запущен на порту %s", cfg.ServerAddress)
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatalf("Ошибка при запуске сервера: %v", err)
	}
}
