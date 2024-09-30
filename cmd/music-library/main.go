package main

import (
	"database/sql"
	"errors"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
	"musicLibrary-Golang-REST/internal/handler"
	"musicLibrary-Golang-REST/internal/repository"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	_ "musicLibrary-Golang-REST/docs"
)

// @title Music Library API
// @version 1.0
// @description API для управления библиотекой песен
// @host localhost:8002
// @BasePath /
func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	gin.SetMode(gin.ReleaseMode)

	logrus.Info("Загрузка конфигурации из .env файла")
	err := godotenv.Load()
	if err != nil {
		logrus.Warn("Не удалось загрузить .env файл. Используются переменные окружения")
	} else {
		logrus.Info(".env файл успешно загружен")
	}

	logrus.Info("Подключение к базе данных...")
	db, err := repository.NewPostgresDB()
	if err != nil {
		logrus.WithError(err).Fatal("Не удалось подключиться к базе данных")
	} else {
		logrus.Info("Успешное подключение к базе данных")
	}

	sqlDB := db.DB

	runMigrations(sqlDB)

	logrus.Info("Инициализация маршрутов API")
	r := gin.Default()

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	h := handler.NewHandler(db)
	h.InitRoutes(r)

	logrus.Info("Запуск сервера на порту: ", os.Getenv("SERVER_PORT"))
	if err := r.Run(":" + os.Getenv("SERVER_PORT")); err != nil {
		logrus.WithError(err).Fatal("Не удалось запустить сервер")
	}
}

func runMigrations(db *sql.DB) {
	logrus.Info("Запуск миграций базы данных...")
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		logrus.WithError(err).Fatal("Ошибка при создании драйвера миграции")
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://./schema",
		"postgres", driver)
	if err != nil {
		logrus.WithError(err).Fatal("Ошибка при создании миграции")
	}

	logrus.Debug("Применение миграций...")
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logrus.WithError(err).Fatal("Ошибка применения миграции")
	}

	logrus.Info("Миграции успешно применены!")
}
