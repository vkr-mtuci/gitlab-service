package main

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/vkr-mtuci/gitlab-service/config"
	"github.com/vkr-mtuci/gitlab-service/internal/adapter"
	"github.com/vkr-mtuci/gitlab-service/internal/handler"
	"github.com/vkr-mtuci/gitlab-service/internal/service"
)

func main() {
	// Настроим zerolog
	output := zerolog.ConsoleWriter{Out: os.Stdout}
	logger := zerolog.New(output).With().Timestamp().Logger()

	// Загружаем конфигурацию
	cfg := config.LoadConfig()

	// Вывод информации о запуске сервиса
	logger.Info().Msg("📢 Запуск GitLab-сервиса...")

	// Создаем клиента для GitLab
	gitLabClient := adapter.NewGitLabClient(cfg)

	// Создаем сервис GitLab
	gitLabService := service.NewGitLabService(gitLabClient)

	// Создаем HTTP-обработчик
	gitLabHandler := handler.NewGitLabHandler(gitLabService)

	// Создаем приложение Fiber
	app := fiber.New()

	// Проверка запуска сервиса
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "✅ GitLab-service is running"})
	})

	// ✅ Регистрируем маршруты
	app.Get("/environments", gitLabHandler.GetEnvironments)                     // Получить список окружений
	app.Get("/environments/:id", gitLabHandler.GetEnvironmentDetails)           // Получить детали окружения
	app.Get("/commits/:ref/:sha", gitLabHandler.GetCommitsInBuild)              // Получить коммиты сборки
	app.Get("/pipelines/:pipeline_id/deploy-jobs", gitLabHandler.GetDeployJobs) // Получить deploy-джобы
	app.Post("/jobs/:job_id/play", gitLabHandler.TriggerDeployJob)              // ✅ Запуск deploy-джобы

	// Запускаем сервер
	logger.Info().Msgf("🚀 Сервис запущен на порту %s", cfg.ServerPort)
	err := app.Listen(":" + cfg.ServerPort)
	if err != nil {
		logger.Fatal().Err(err).Msg("❌ Ошибка запуска сервера")
	}
}
