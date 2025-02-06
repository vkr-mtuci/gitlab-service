package test

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vkr-mtuci/gitlab-service/config"
	"github.com/vkr-mtuci/gitlab-service/internal/adapter"
	"github.com/vkr-mtuci/gitlab-service/internal/service"
	"github.com/vkr-mtuci/gitlab-service/test/mocks"
)

func TestMock_TriggerDeployJob(t *testing.T) {
	mockClient := &mocks.MockGitLabClient{} // Используем мок-клиент вместо реального HTTP-запроса

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	job, err := mockClient.TriggerDeployJob(ctx, "7")

	require.NoError(t, err)
	assert.NotNil(t, job)
	assert.Equal(t, 7, job.ID)
	assert.Equal(t, "pending", job.Status)
	assert.Equal(t, "deploy-production", job.Name)
	assert.Equal(t, "deploy", job.Stage)
	assert.Equal(t, "https://example.com/foo/bar/-/jobs/7", job.WebURL)
}

func TestGetPipelineJobs_EmptyResponse(t *testing.T) {
	mockServer := mocks.NewMockGitLabServer()
	defer mockServer.Close()

	cfg := &config.Config{
		GitLabBaseURL:   mockServer.URL,
		GitLabAPIURL:    "/api/v4/projects/",
		GitLabAPIToken:  "test-token",
		GitLabProjectID: "1",
	}

	client := adapter.NewGitLabClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 🔧 Устанавливаем мок-ответ: пустой массив джобов
	mockServer.SetResponse("/api/v4/projects/1/pipelines/6/jobs", 200, `[]`)

	// Вызываем метод
	jobs, err := client.GetPipelineJobs(ctx, "6")

	// ✅ Проверяем, что ошибки нет
	require.NoError(t, err)

	// ✅ Проверяем, что jobs пустой
	assert.Empty(t, jobs, "Ожидался пустой список джоб")
}

func TestGetPipelineJobs_ServerError(t *testing.T) {
	mockServer := mocks.NewMockGitLabServer()
	defer mockServer.Close()

	cfg := &config.Config{
		GitLabBaseURL:   mockServer.URL,
		GitLabAPIURL:    "/api/v4/projects/",
		GitLabAPIToken:  "test-token",
		GitLabProjectID: "1",
	}

	client := adapter.NewGitLabClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Эмулируем ошибку 500
	mockServer.SetErrorResponse("/api/v4/projects/1/pipelines/6/jobs", 500, "Internal Server Error")

	jobs, err := client.GetPipelineJobs(ctx, "6")

	require.Error(t, err)
	assert.Nil(t, jobs)
}

// ✅ Тест успешного запуска deploy-джобы
func TestTriggerDeployJob_Success(t *testing.T) {
	mockServer := mocks.NewMockGitLabServer()
	defer mockServer.Close()

	cfg := &config.Config{
		GitLabBaseURL:   mockServer.URL,
		GitLabAPIURL:    "/api/v4/projects/",
		GitLabAPIToken:  "test-token",
		GitLabProjectID: "1",
	}

	client := adapter.NewGitLabClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	job, err := client.TriggerDeployJob(ctx, "7")

	require.NoError(t, err)
	assert.NotNil(t, job)
	assert.Equal(t, 7, job.ID)
	assert.Equal(t, "pending", job.Status)
	assert.Equal(t, "https://example.com/foo/bar/-/jobs/7", job.WebURL)
}

// ❌ Тест ошибки при пустом `job_id`
func TestTriggerDeployJob_InvalidJobID(t *testing.T) {
	mockServer := mocks.NewMockGitLabServer()
	defer mockServer.Close()

	cfg := &config.Config{
		GitLabBaseURL:   mockServer.URL,
		GitLabAPIURL:    "/api/v4/projects/",
		GitLabAPIToken:  "test-token",
		GitLabProjectID: "1",
	}

	client := adapter.NewGitLabClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	job, err := client.TriggerDeployJob(ctx, "")

	assert.Error(t, err)
	assert.Nil(t, job)
}

// ❌ Тест ошибки 404 (job не найдена)
func TestTriggerDeployJob_NotFound(t *testing.T) {
	mockServer := mocks.NewMockGitLabServer()
	defer mockServer.Close()

	mockServer.SetErrorResponse("/api/v4/projects/1/jobs/999/play", 404, `{"message": "404 job not found"}`)

	cfg := &config.Config{
		GitLabBaseURL:   mockServer.URL,
		GitLabAPIURL:    "/api/v4/projects/",
		GitLabAPIToken:  "test-token",
		GitLabProjectID: "1",
	}

	client := adapter.NewGitLabClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	job, err := client.TriggerDeployJob(ctx, "999")

	assert.Error(t, err)
	assert.Nil(t, job)
	assert.Contains(t, err.Error(), "404 job not found")
}

// ❌ Тест ошибки 500 (внутренняя ошибка сервера)
func TestTriggerDeployJob_ServerError(t *testing.T) {
	mockServer := mocks.NewMockGitLabServer()
	defer mockServer.Close()

	// Эмулируем 500-ю ошибку от сервера при запуске job 7
	mockServer.SetErrorResponse("/api/v4/projects/1/jobs/7/play", 500, `{"message": "Internal Server Error"}`)

	cfg := &config.Config{
		GitLabBaseURL:   mockServer.URL,
		GitLabAPIURL:    "/api/v4/projects/",
		GitLabAPIToken:  "test-token",
		GitLabProjectID: "1",
	}

	client := adapter.NewGitLabClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Вызываем метод, ожидая ошибку
	job, err := client.TriggerDeployJob(ctx, "7")

	// Ожидаем ошибку
	assert.Error(t, err)
	assert.Nil(t, job)
	assert.Contains(t, err.Error(), "Internal Server Error")
}

func TestGetDeployJobs_Success(t *testing.T) {
	mockServer := mocks.NewMockGitLabServer()
	defer mockServer.Close()

	cfg := &config.Config{
		GitLabBaseURL:   mockServer.URL,
		GitLabAPIURL:    "/api/v4/projects/",
		GitLabAPIToken:  "test-token",
		GitLabProjectID: "1",
	}

	client := adapter.NewGitLabClient(cfg)
	service := service.NewGitLabService(client)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Вызываем метод
	jobs, err := service.GetDeployJobs(ctx, "6")

	require.NoError(t, err)
	assert.Len(t, jobs, 1) // ✅ Ожидаем 1 джобу в deploy-стадии

	// Проверяем корректность данных
	assert.Equal(t, 7, jobs[0].ID)
	assert.Equal(t, "failed", jobs[0].Status)
	assert.Equal(t, "https://example.com/foo/bar/-/jobs/7", jobs[0].WebURL)
}

func TestMockGitLabServer_HandleRequest(t *testing.T) {
	mockServer := mocks.NewMockGitLabServer()
	defer mockServer.Close()

	// ✅ Устанавливаем кастомный ответ для теста
	mockServer.SetResponse("/api/v4/custom-test", 200, `{"message": "custom response"}`)

	resp, err := http.Get(mockServer.URL + "/api/v4/custom-test")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"message": "custom response"}`, string(body))
}

func TestMockGitLabServer_HandleRequest_NotFound(t *testing.T) {
	mockServer := mocks.NewMockGitLabServer()
	defer mockServer.Close()

	resp, err := http.Get(mockServer.URL + "/api/v4/unknown-endpoint")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"message": "404 page not found"}`, string(body))
}

func TestParseGitLabError_JSONError(t *testing.T) {
	resp := []byte(`{"message":"Unauthorized"}`)
	err := adapter.ParseGitLabError(resp)
	assert.EqualError(t, err, "GitLab API Error: Unauthorized")
}

func TestParseGitLabError_NonJSON(t *testing.T) {
	resp := []byte(`This is not JSON`)
	err := adapter.ParseGitLabError(resp)
	assert.EqualError(t, err, "GitLab API: This is not JSON")
}

func TestHandleRequest_CustomErrorResponse(t *testing.T) {
	mockServer := mocks.NewMockGitLabServer()
	defer mockServer.Close()

	// Устанавливаем кастомную ошибку
	mockServer.SetErrorResponse("/api/v4/projects/1/environments", 500, `{"error":"Internal Server Error"}`)

	client := adapter.NewGitLabClient(&config.Config{
		GitLabBaseURL:   mockServer.URL,
		GitLabAPIURL:    "/api/v4/projects/",
		GitLabAPIToken:  "test-token",
		GitLabProjectID: "1",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	environments, err := client.GetEnvironments(ctx)
	assert.Error(t, err)
	assert.Nil(t, environments)
}

func TestGetEnvironments_APIError(t *testing.T) {
	mockServer := mocks.NewMockGitLabServer()
	defer mockServer.Close()

	cfg := &config.Config{
		GitLabBaseURL:   mockServer.URL,
		GitLabAPIURL:    "/api/v4/projects/",
		GitLabAPIToken:  "test-token",
		GitLabProjectID: "1",
	}

	client := adapter.NewGitLabClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ❌ Эмулируем ошибку 500 от сервера
	mockServer.SetErrorResponse("/api/v4/projects/1/environments", 500, "Internal Server Error")

	environments, err := client.GetEnvironments(ctx)
	assert.Error(t, err)
	assert.Nil(t, environments)
}

func TestGetEnvironmentDetails_404(t *testing.T) {
	mockServer := mocks.NewMockGitLabServer()
	defer mockServer.Close()

	mockServer.SetErrorResponse("/api/v4/projects/1/environments/1", 404, `{"message":"Not Found"}`)

	client := adapter.NewGitLabClient(&config.Config{
		GitLabBaseURL:   mockServer.URL,
		GitLabAPIURL:    "/api/v4/projects/",
		GitLabAPIToken:  "test-token",
		GitLabProjectID: "1",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	details, err := client.GetEnvironmentDetails(ctx, "1")
	assert.Error(t, err)
	assert.Nil(t, details)
}

func TestGetEnvironmentDetails_EmptyResponse(t *testing.T) {
	mockServer := mocks.NewMockGitLabServer()
	defer mockServer.Close()

	cfg := &config.Config{
		GitLabBaseURL:   mockServer.URL,
		GitLabAPIURL:    "/api/v4/projects/",
		GitLabAPIToken:  "test-token",
		GitLabProjectID: "1",
	}

	client := adapter.NewGitLabClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mockServer.SetResponse("/api/v4/projects/1/environments/1", 200, `[]`)

	environment, err := client.GetEnvironmentDetails(ctx, "1")
	assert.Error(t, err)
	assert.Nil(t, environment)
}

func TestGetBuildVersion_NotFound(t *testing.T) {
	mockServer := mocks.NewMockGitLabServer()
	defer mockServer.Close()

	mockServer.SetResponse("/api/v4/projects/1/jobs/201/trace", 200, `No build version here`)

	client := adapter.NewGitLabClient(&config.Config{
		GitLabBaseURL:   mockServer.URL,
		GitLabAPIURL:    "/api/v4/projects/",
		GitLabAPIToken:  "test-token",
		GitLabProjectID: "1",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	version, err := client.GetBuildVersion(ctx, "201")
	assert.Error(t, err)
	assert.Empty(t, version)
}

func TestGetCommitsBetweenSHAs_NoCommits(t *testing.T) {
	mockServer := mocks.NewMockGitLabServer()
	defer mockServer.Close()

	cfg := &config.Config{
		GitLabBaseURL:   mockServer.URL,
		GitLabAPIURL:    "/api/v4/projects/",
		GitLabAPIToken:  "test-token",
		GitLabProjectID: "1",
		JiraProject:     "JIRA",
	}

	client := adapter.NewGitLabClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mockServer.SetResponse("/api/v4/projects/1/repository/commits?ref_name=develop", 200, "[]")

	commits, err := client.GetCommitsBetweenSHAs(ctx, "develop", "sha-123", "sha-124")
	assert.Error(t, err)
	assert.Nil(t, commits)
}

// ✅ Тест получения списка окружений
func TestGetEnvironments_Success(t *testing.T) {
	mockServer := mocks.NewMockGitLabServer()
	defer mockServer.Close()

	cfg := &config.Config{
		GitLabBaseURL:   mockServer.URL,
		GitLabAPIURL:    "/api/v4/projects/",
		GitLabAPIToken:  "test-token",
		GitLabProjectID: "1",
	}

	client := adapter.NewGitLabClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	environments, err := client.GetEnvironments(ctx)
	require.NoError(t, err)
	assert.Len(t, environments, 2)
	assert.Equal(t, "staging", environments[0].Name)
}

// ✅ Тест получения информации об окружении
func TestGetEnvironmentDetails_Success(t *testing.T) {
	mockServer := mocks.NewMockGitLabServer()
	defer mockServer.Close()

	cfg := &config.Config{
		GitLabBaseURL:   mockServer.URL,
		GitLabAPIURL:    "/api/v4/projects/",
		GitLabAPIToken:  "test-token",
		GitLabProjectID: "1",
	}

	client := adapter.NewGitLabClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	details, err := client.GetEnvironmentDetails(ctx, "1")
	require.NoError(t, err)
	assert.NotNil(t, details)
	assert.Equal(t, "staging", details.EnvironmentName)
	assert.Equal(t, "success", details.DeployStatus)
	assert.Equal(t, "1.2.3", details.BuildVersion)
}

// ✅ Тест поиска Jira-ключей
func TestExtractJiraKeys(t *testing.T) {
	commits := []adapter.CommitInfo{
		{Message: "Fix bug JIRA-123"},
		{Message: "Feature added JIRA-456"},
		{Message: "No Jira key here"},
	}

	keys := adapter.ExtractJiraKeys(commits, "JIRA")
	assert.Len(t, keys, 2)
	assert.Contains(t, keys, "JIRA-123")
	assert.Contains(t, keys, "JIRA-456")
}

// ✅ Тест получения списка коммитов между SHA
func TestGetCommitsBetweenSHAs_Success(t *testing.T) {
	mockServer := mocks.NewMockGitLabServer()
	defer mockServer.Close()

	cfg := &config.Config{
		GitLabBaseURL:   mockServer.URL,
		GitLabAPIURL:    "/api/v4/projects/",
		GitLabAPIToken:  "test-token",
		GitLabProjectID: "1",
		JiraProject:     "JIRA",
	}

	client := adapter.NewGitLabClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	commits, err := client.GetCommitsBetweenSHAs(ctx, "develop", "commit-3", "commit-1")
	require.NoError(t, err)
	assert.Len(t, commits, 2) // ✅ Ожидаем 2 коммита

	// ✅ Проверяем правильные SHA
	assert.Equal(t, "commit-1", commits[0].ID)
	assert.Equal(t, "commit-2", commits[1].ID)

	// ✅ Проверяем Jira-ключи
	assert.Contains(t, commits[0].JiraKeys, "JIRA-123")
	assert.Contains(t, commits[1].JiraKeys, "JIRA-456")
}

// ✅ Тест получения версии билда из логов
func TestGetBuildVersion_Success(t *testing.T) {
	mockServer := mocks.NewMockGitLabServer()
	defer mockServer.Close()

	cfg := &config.Config{
		GitLabBaseURL:   mockServer.URL,
		GitLabAPIURL:    "/api/v4/projects/",
		GitLabAPIToken:  "test-token",
		GitLabProjectID: "1",
	}

	client := adapter.NewGitLabClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	buildVersion, err := client.GetBuildVersion(ctx, "201")
	require.NoError(t, err)
	assert.Equal(t, "1.2.3", buildVersion)
}

// ✅ Тест обработки ошибок GitLab API
func TestParseGitLabError(t *testing.T) {
	err := adapter.ParseGitLabError([]byte(`{"message": "Some error occurred"}`))
	assert.Error(t, err)
	assert.Equal(t, "GitLab API Error: Some error occurred", err.Error())
}

// TestGetPreviousPipelineSHA_Success проверяет успешное получение SHA предыдущего пайплайна
func TestGetPreviousPipelineSHA_Success(t *testing.T) {
	mockServer := mocks.NewMockGitLabServer() // Запускаем мок-сервер
	defer mockServer.Close()

	// Загружаем конфигурацию с мок-URL
	cfg := &config.Config{
		GitLabBaseURL:   mockServer.URL,
		GitLabAPIURL:    "/api/v4/projects/",
		GitLabAPIToken:  "test-token",
		GitLabProjectID: "1",
	}

	client := adapter.NewGitLabClient(cfg)

	// Проверяем получение предыдущего SHA
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	prevSHA, err := client.GetPreviousPipelineSHA(ctx, "develop", "sha-123")
	require.NoError(t, err)
	assert.Equal(t, "sha-122", prevSHA)
}

// TestGetPreviousPipelineSHA_NotFound проверяет случай, когда предыдущий SHA не найден
func TestGetPreviousPipelineSHA_NotFound(t *testing.T) {
	mockServer := mocks.NewMockGitLabServer()
	defer mockServer.Close()

	cfg := &config.Config{
		GitLabBaseURL:   mockServer.URL,
		GitLabAPIURL:    "/api/v4/projects/",
		GitLabAPIToken:  "test-token",
		GitLabProjectID: "1",
	}

	client := adapter.NewGitLabClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	prevSHA, err := client.GetPreviousPipelineSHA(ctx, "develop", "unknown-sha")
	assert.Error(t, err)
	assert.Empty(t, prevSHA)
}
