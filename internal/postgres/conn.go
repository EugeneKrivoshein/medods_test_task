package postgres

import (
	"database/sql"
	"fmt"

	"github.com/EugeneKrivoshein/medods_test_task/config"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type PostgresProvider struct {
	db *sql.DB
}

func (p *PostgresProvider) DB() *sql.DB {
	return p.db
}

func (p *PostgresProvider) Close() error {
	return p.db.Close()
}

func NewPostgresProvider() (*PostgresProvider, error) {
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	cfg, err := config.LoadConfig("config.env")

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBPort, cfg.DBName,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Errorf("Ошибка подключения к базе данных: %v", err)
		return nil, fmt.Errorf("ошибка подключения к базе данных: %w", err)
	}

	if err := db.Ping(); err != nil {
		log.Errorf("Ошибка при проверке подключения: %v", err)
		return nil, fmt.Errorf("база данных недоступна: %w", err)
	}

	log.Println("Successfully connected to the database.")

	return &PostgresProvider{db: db}, nil
}
