package test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vkr-mtuci/gitlab-service/config"
)

func TestLoadConfig(t *testing.T) {
	// ✅ Устанавливаем переменные окружения
	os.Setenv("SERVER_PORT", "8080")
	os.Setenv("GITLAB_BASE_URL", "https://gitlab.example.com")
	os.Setenv("GITLAB_API_URL", "/api/v4/projects/")
	os.Setenv("GITLAB_API_TOKEN", "dummy-token")
	os.Setenv("GITLAB_PROJECT_ID", "123")
	os.Setenv("JIRA_PROJECT", "JIRA")

	cfg := config.LoadConfig()

	// ✅ Проверяем корректность конфигурации
	assert.Equal(t, "8080", cfg.ServerPort)
	assert.Equal(t, "https://gitlab.example.com", cfg.GitLabBaseURL)
	assert.Equal(t, "/api/v4/projects/", cfg.GitLabAPIURL)
	assert.Equal(t, "dummy-token", cfg.GitLabAPIToken)
	assert.Equal(t, "123", cfg.GitLabProjectID)
	assert.Equal(t, "JIRA", cfg.JiraProject)
}
