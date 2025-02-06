package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config структура для хранения конфигурации приложения
type Config struct {
	ServerPort      string
	GitLabBaseURL   string
	GitLabAPIURL    string
	GitLabAPIToken  string
	GitLabProjectID string
	JiraProject     string
}

// LoadConfig загружает переменные окружения в структуру Config
func LoadConfig() *Config {
	_ = godotenv.Load() // Загружаем переменные окружения из .env (если файл есть)

	config := &Config{
		ServerPort:      os.Getenv("SERVER_PORT"),
		GitLabBaseURL:   os.Getenv("GITLAB_BASE_URL"),
		GitLabAPIURL:    os.Getenv("GITLAB_API_URL"),
		GitLabAPIToken:  os.Getenv("GITLAB_API_TOKEN"),
		GitLabProjectID: os.Getenv("GITLAB_PROJECT_ID"),
		JiraProject:     os.Getenv("JIRA_PROJECT"),
	}

	// Проверяем, заданы ли критически важные переменные
	if config.GitLabBaseURL == "" || config.GitLabAPIURL == "" || config.GitLabAPIToken == "" || config.GitLabProjectID == "" || config.JiraProject == "" {
		log.Fatal("❌ Ошибка: Не заданы все обязательные переменные окружения для GitLab")
	}

	return config
}
