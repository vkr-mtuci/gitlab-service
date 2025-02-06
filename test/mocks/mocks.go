package mocks

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/mock"
	"github.com/vkr-mtuci/gitlab-service/internal/adapter"
)

// MockGitLabClient - мок-реализация клиента GitLab
type MockGitLabClient struct {
	mock.Mock
}

// MockGitLabServer - структура мок-сервера с кастомными ответами
type MockGitLabServer struct {
	*httptest.Server
	mu        sync.Mutex
	responses map[string]mockResponse
}

// mockResponse - структура для хранения кастомного ответа API
type mockResponse struct {
	status int
	body   string
}

// GetEnvironments - возвращает тестовые окружения
func (m *MockGitLabClient) GetEnvironments(ctx context.Context) ([]adapter.Environment, error) {
	return []adapter.Environment{
		{ID: 1, Name: "staging"},
		{ID: 2, Name: "production"},
	}, nil
}

// GetEnvironmentDetails - возвращает тестовую информацию об окружении
func (m *MockGitLabClient) GetEnvironmentDetails(ctx context.Context, environmentID string) (*adapter.DeploymentInfo, error) {
	if environmentID == "1" {
		return &adapter.DeploymentInfo{
			EnvironmentName: "staging",
			DeploymentDate:  "2024-02-10T12:00:00Z",
			PipelineID:      101,
			PipelineURL:     "https://gitlab.example.com/pipelines/101",
			JobID:           201,
			JobURL:          "https://gitlab.example.com/jobs/201",
			DeployStatus:    "success",
			BuildVersion:    "1.2.3",
		}, nil
	}
	return nil, errors.New("environment not found")
}

// GetPreviousPipelineSHA - возвращает SHA предыдущего пайплайна
func (m *MockGitLabClient) GetPreviousPipelineSHA(ctx context.Context, ref, currentSHA string) (string, error) {
	if currentSHA == "sha-123" {
		return "sha-122", nil
	}
	return "", errors.New("previous SHA not found")
}

// GetCommitsBetweenSHAs - возвращает тестовые коммиты между SHA
func (m *MockGitLabClient) GetCommitsBetweenSHAs(ctx context.Context, ref, fromSHA, toSHA string) ([]adapter.CommitInfo, error) {
	if fromSHA == "sha-122" && toSHA == "sha-123" {
		return []adapter.CommitInfo{
			{
				ID:          "commit-1",
				CreatedAt:   "2025-02-06T12:34:56Z",
				Message:     "Fix bug JIRA-123",
				AuthorName:  "Test User",
				AuthorEmail: "test@example.com",
				WebURL:      "https://gitlab.example.com/commit/commit-1",
				JiraKeys:    []string{"JIRA-123"},
			},
			{
				ID:          "commit-2",
				CreatedAt:   "2025-02-06T12:40:56Z",
				Message:     "Feature added JIRA-456",
				AuthorName:  "Dev Tester",
				AuthorEmail: "dev@example.com",
				WebURL:      "https://gitlab.example.com/commit/commit-2",
				JiraKeys:    []string{"JIRA-456"},
			},
		}, nil
	}
	return nil, errors.New("no commits found")
}

// GetPipelineJobs - возвращает тестовые джобы для пайплайна
func (m *MockGitLabClient) GetPipelineJobs(ctx context.Context, pipelineID string) ([]adapter.JobInfo, error) {
	// Возвращаем фиктивные джобы со stage=deploy
	if pipelineID == "9679696" {
		finishedAt1, _ := time.Parse(time.RFC3339, "2025-02-06T12:40:56Z")
		finishedAt2, _ := time.Parse(time.RFC3339, "2025-02-06T12:50:10Z")
		return []adapter.JobInfo{
			{
				ID:         1001,
				Stand:      "deploy to production",
				Stage:      "deploy",
				Status:     "success",
				WebURL:     "https://gitlab.example.com/job/101",
				FinishedAt: finishedAt1,
			},
			{
				ID:         1002,
				Stand:      "deploy to staging",
				Stage:      "deploy",
				Status:     "failed",
				WebURL:     "https://gitlab.example.com/job/102",
				FinishedAt: finishedAt2,
			},
		}, nil
	}
	return nil, errors.New("❌ Джобы не найдены для pipelineID=" + pipelineID)
}

// NewMockGitLabServer создаёт тестовый сервер, который эмулирует GitLab API
func NewMockGitLabServer() *MockGitLabServer {
	mock := &MockGitLabServer{
		responses: make(map[string]mockResponse),
	}

	handler := http.NewServeMux()

	// ✅ Мокируем ответ на GET /projects/:id/jobs/:job_id/trace
	handler.HandleFunc("/api/v4/projects/1/jobs/201/trace", func(w http.ResponseWriter, r *http.Request) {
		log.Debug().Msg("📡 Запрос логов job 201")

		mock.mu.Lock()
		resp, exists := mock.responses["/api/v4/projects/1/jobs/201/trace"]
		mock.mu.Unlock()

		if exists {
			w.WriteHeader(resp.status)
			w.Write([]byte(resp.body))
			return
		}

		// ✅ Стандартный успешный ответ с BUILD_VERSION
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`BUILD_VERSION=1.2.3`))
	})

	// ✅ Мокируем ответ на GET /projects/:id/repository/commits?ref_name=develop
	handler.HandleFunc("/api/v4/projects/1/repository/commits", func(w http.ResponseWriter, r *http.Request) {
		ref := r.URL.Query().Get("ref_name")
		log.Debug().Msgf("📡 Запрос коммитов с ref_name=%s", ref)

		if ref == "develop" {
			log.Info().Msg("✅ Возвращаем 3 коммита")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[
				{"id": "commit-1", "message": "Fix bug JIRA-123", "author_name": "Test User", "author_email": "test@example.com", "created_at": "2025-02-06T12:34:56Z", "web_url": "https://gitlab.example.com/commit/commit-1"},
				{"id": "commit-2", "message": "Feature added JIRA-456", "author_name": "Dev Tester", "author_email": "dev@example.com", "created_at": "2025-02-06T12:30:00Z", "web_url": "https://gitlab.example.com/commit/commit-2"},
				{"id": "commit-3", "message": "Feature added JIRA-789", "author_name": "Dev Tester", "author_email": "dev@example.com", "created_at": "2025-02-06T12:25:00Z", "web_url": "https://gitlab.example.com/commit/commit-3"}
			]`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	})

	// ✅ Мокируем ответ на GET /projects/:id/pipelines
	handler.HandleFunc("/api/v4/projects/1/pipelines", func(w http.ResponseWriter, r *http.Request) {
		ref := r.URL.Query().Get("ref")
		if ref == "develop" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id": 1001, "sha": "sha-123"}, {"id": 1000, "sha": "sha-122"}]`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	})

	// ✅ Мокируем ответ на GET /projects/:id/environments
	handler.HandleFunc("/api/v4/projects/1/environments", func(w http.ResponseWriter, r *http.Request) {
		log.Debug().Msg("📡 Запрос окружений")

		// 🔥 Если кастомный ответ установлен через SetErrorResponse, используем его!
		mock.mu.Lock()
		resp, exists := mock.responses["/api/v4/projects/1/environments"]
		mock.mu.Unlock()

		if exists {
			w.WriteHeader(resp.status)
			w.Write([]byte(resp.body))
			return
		}

		// ✅ Стандартный ответ, если кастомный не подставлен
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{"id": 1, "name": "staging"},
			{"id": 2, "name": "production"}
		]`))
	})

	// ✅ Мокируем ответ на GET /projects/:id/environments/:env_id
	handler.HandleFunc("/api/v4/projects/1/environments/1", func(w http.ResponseWriter, r *http.Request) {
		log.Debug().Msg("📡 Запрос информации по окружению 1")

		// 🔥 Если есть кастомный ответ через SetErrorResponse, используем его
		mock.mu.Lock()
		resp, exists := mock.responses["/api/v4/projects/1/environments/1"]
		mock.mu.Unlock()

		if exists {
			w.WriteHeader(resp.status)
			w.Write([]byte(resp.body))
			return
		}

		// ✅ Стандартный успешный ответ
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": 1, "name": "staging", "external_url": "https://staging.example.com", "created_at": "2024-02-10T12:00:00Z",
			"last_deployment": {
				"created_at": "2024-02-11T10:00:00Z", "ref": "main", "sha": "abc123",
				"deployable": {"id": 201, "web_url": "https://gitlab.example.com/jobs/201", "status": "success",
					"pipeline": {"id": 101, "web_url": "https://gitlab.example.com/pipelines/101"}
				}
			}
		}`))
	})

	// ✅ Добавляем поддержку кастомных ответов для jobs
	handler.HandleFunc("/api/v4/projects/1/pipelines/6/jobs", func(w http.ResponseWriter, r *http.Request) {
		log.Debug().Msg("📡 Запрос списка джоб для пайплайна 6")

		// Проверяем, есть ли кастомный ответ
		mock.mu.Lock()
		resp, exists := mock.responses[r.URL.Path]
		mock.mu.Unlock()

		if exists {
			w.WriteHeader(resp.status)
			w.Write([]byte(resp.body))
			return
		}

		// Если кастомный ответ не задан, возвращаем стандартный список (для других тестов)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"id": 7, "status": "failed", "finished_at": "2025-02-06T17:54:27.895Z", "stage": "deploy", "web_url": "https://example.com/foo/bar/-/jobs/7"}]`))
	})

	handler.HandleFunc("/api/v4/projects/1/jobs/7/play", func(w http.ResponseWriter, r *http.Request) {
		mock.mu.Lock()
		defer mock.mu.Unlock()

		// Проверяем, есть ли принудительно установленный ответ (например, 500)
		if resp, exists := mock.responses[r.URL.Path]; exists {
			w.WriteHeader(resp.status)
			w.Write([]byte(resp.body))
			return
		}

		// Если ошибки нет, возвращаем стандартный успешный ответ
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"error": "Method Not Allowed"}`))
			return
		}

		log.Debug().Msg("🚀 Мок: Запуск deploy-джобы job 7")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": 7,
			"name": "deploy-production",
			"stage": "deploy",
			"status": "pending",
			"created_at": "2025-02-06T20:50:00Z",
			"web_url": "https://example.com/foo/bar/-/jobs/7"
		}`))
	})

	// Добавляем обработку динамических ошибок
	handler.HandleFunc("/", mock.handleRequest)

	mock.Server = httptest.NewServer(handler)
	return mock
}

// TriggerDeployJob - мок для запуска деплоя
func (m *MockGitLabClient) TriggerDeployJob(ctx context.Context, jobID string) (*adapter.TriggeredJob, error) {
	if jobID == "7" {
		return &adapter.TriggeredJob{
			ID:        7,
			Name:      "deploy-production",
			Stage:     "deploy",
			Status:    "pending",
			CreatedAt: time.Now(),
			WebURL:    "https://example.com/foo/bar/-/jobs/7",
		}, nil
	}
	return nil, errors.New("job not found")
}

// handleRequest - обрабатывает запросы и подставляет кастомные ответы
func (m *MockGitLabServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Если для пути есть кастомный ответ - возвращаем его
	if resp, exists := m.responses[r.URL.Path]; exists {
		w.WriteHeader(resp.status)
		w.Write([]byte(resp.body))
		return
	}

	// По умолчанию 404
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"message": "404 page not found"}`))
}

// SetErrorResponse - устанавливает кастомную ошибку для конкретного пути
func (m *MockGitLabServer) SetErrorResponse(path string, status int, body string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[path] = mockResponse{status, body}
}

// SetResponse - устанавливает кастомный ответ для пути
func (m *MockGitLabServer) SetResponse(path string, status int, body string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[path] = mockResponse{status, body}
}
