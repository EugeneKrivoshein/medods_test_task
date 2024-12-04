package migrations

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/EugeneKrivoshein/medods_test_task/internal/postgres"
	"github.com/sirupsen/logrus"
)

func RunMigrations(provider *postgres.PostgresProvider, migrationPath string) error {
	if err := createMigrationsTableIfNotExist(provider.DB()); err != nil {
		return fmt.Errorf("ошибка создания таблицы для миграций: %v", err)
	}

	migrationFiles, err := os.ReadDir(migrationPath)
	if err != nil {
		return fmt.Errorf("ошибка чтения файлов миграций: %v", err)
	}

	var upFiles []string
	for _, file := range migrationFiles {
		if strings.HasSuffix(file.Name(), ".up.sql") {
			upFiles = append(upFiles, filepath.Join(migrationPath, file.Name()))
		}
	}

	for _, file := range upFiles {
		if isMigrationApplied(provider.DB(), file) {
			logrus.Infof("Миграция %s уже была выполнена, пропускаем", file)
			continue
		}

		if err := applyMigration(provider, file); err != nil {
			return fmt.Errorf("ошибка выполнения миграции %s: %v", file, err)
		}

		if err := recordMigration(provider.DB(), file); err != nil {
			return fmt.Errorf("ошибка записи выполненной миграции %s: %v", file, err)
		}

		logrus.Infof("Миграция %s выполнена успешно", file)
	}

	return nil
}

func createMigrationsTableIfNotExist(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id SERIAL PRIMARY KEY,
			migration_name TEXT NOT NULL UNIQUE,
			applied_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := db.Exec(query)
	return err
}

func isMigrationApplied(db *sql.DB, migrationFile string) bool {
	var count int
	query := `SELECT COUNT(*) FROM schema_migrations WHERE migration_name = $1`
	err := db.QueryRow(query, migrationFile).Scan(&count)
	if err != nil {
		logrus.Errorf("Ошибка при проверке миграции %s: %v", migrationFile, err)
		return false
	}
	return count > 0
}

func applyMigration(provider *postgres.PostgresProvider, migrationFile string) error {
	sqlBytes, err := os.ReadFile(migrationFile)
	if err != nil {
		return fmt.Errorf("ошибка чтения миграции из файла %s: %v", migrationFile, err)
	}

	_, err = provider.DB().Exec(string(sqlBytes))
	if err != nil {
		return fmt.Errorf("ошибка выполнения миграции %s: %v", migrationFile, err)
	}

	return nil
}

func recordMigration(db *sql.DB, migrationFile string) error {
	query := `INSERT INTO schema_migrations (migration_name) VALUES ($1)`
	_, err := db.Exec(query, migrationFile)
	return err
}
