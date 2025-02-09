package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
	"github.com/vkr-mtuci/gitlab-service/config"
)

// GitLabClientInterface - интерфейс для моков
type GitLabClientInterface interface {
	GetEnvironments(ctx context.Context) ([]Environment, error)
	GetEnvironmentDetails(ctx context.Context, environmentID string) (*DeploymentInfo, error)
	GetPreviousPipelineSHA(ctx context.Context, ref, currentSHA string) (string, error)
	GetCommitsBetweenSHAs(ctx context.Context, ref, fromSHA, toSHA string) ([]CommitInfo, error)
	GetPipelineJobs(ctx context.Context, pipelineID string) ([]JobInfo, error)
	TriggerDeployJob(ctx context.Context, jobID string) (*TriggeredJob, error) // ✅ Новый метод
}

// Убедимся, что GitLabClient реализует интерфейс GitLabClientInterface
var _ GitLabClientInterface = (*GitLabClient)(nil)

// GitLabClient - клиент для взаимодействия с API GitLab
type GitLabClient struct {
	client      *resty.Client
	baseURL     string
	apiURL      string
	projectID   string
	jiraProject string
}

// NewGitLabClient - создание нового клиента для GitLab
func NewGitLabClient(cfg *config.Config) *GitLabClient {
	client := resty.New().
		SetBaseURL(cfg.GitLabBaseURL).
		SetTimeout(10*time.Second).
		SetHeader("Accept", "application/json").
		SetHeader("PRIVATE-TOKEN", cfg.GitLabAPIToken) // ✅ Авторизация через PRIVATE-TOKEN

	log.Info().Msg("🔗 Подключение к GitLab API: " + cfg.GitLabBaseURL)

	return &GitLabClient{
		client:      client,
		baseURL:     cfg.GitLabBaseURL,
		apiURL:      cfg.GitLabAPIURL,
		projectID:   cfg.GitLabProjectID,
		jiraProject: cfg.JiraProject,
	}
}

// GetEnvironments - получает список окружений для указанного проекта
func (g *GitLabClient) GetEnvironments(ctx context.Context) ([]Environment, error) {
	if g.projectID == "" {
		return nil, fmt.Errorf("❌ projectID не может быть пустым")
	}

	url := fmt.Sprintf("%s%s%s/environments", g.baseURL, g.apiURL, g.projectID)
	log.Debug().Msgf("📡 Запрос окружений GitLab: projectID=%s, URL=%s", g.projectID, url)

	resp, err := g.client.R().
		SetContext(ctx).
		Get(url)

	if err != nil {
		log.Error().Err(err).Msg("❌ Ошибка запроса окружений GitLab")
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		log.Warn().Msgf("⚠️ GitLab вернул статус %d", resp.StatusCode())
		return nil, ParseGitLabError(resp.Body()) // Теперь функция используется!
	}

	// Распарсим JSON-ответ
	var environments []Environment
	if err := json.Unmarshal(resp.Body(), &environments); err != nil {
		log.Error().Err(err).Msg("❌ Ошибка парсинга окружений GitLab")
		return nil, err
	}

	log.Info().Msgf("✅ Получено %d окружений для проекта %s", len(environments), g.projectID)
	return environments, nil
}

// ParseGitLabError - обрабатывает тело ответа с ошибкой от GitLab API
func ParseGitLabError(body []byte) error {
	var gitlabErr GitLabError

	// ✅ Проверяем, пустое ли тело
	if len(body) == 0 {
		return fmt.Errorf("GitLab API вернул пустой ответ")
	}

	// ✅ Если парсинг JSON неудачен, возвращаем тело как строку
	if err := json.Unmarshal(body, &gitlabErr); err != nil {
		return fmt.Errorf("GitLab API: %s", string(body))
	}

	return &gitlabErr
}

// GetEnvironmentDetails - получает информацию о конкретном окружении
func (g *GitLabClient) GetEnvironmentDetails(ctx context.Context, environmentID string) (*DeploymentInfo, error) {
	if environmentID == "" {
		return nil, fmt.Errorf("❌ environmentID не может быть пустым")
	}

	url := fmt.Sprintf("%s%s%s/environments/%s", g.baseURL, g.apiURL, g.projectID, environmentID)
	log.Debug().Msgf("📡 Запрос информации об окружении: environmentID=%s, URL=%s", environmentID, url)

	resp, err := g.client.R().
		SetContext(ctx).
		Get(url)

	if err != nil {
		log.Error().Err(err).Msg("❌ Ошибка запроса к GitLab")
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		log.Warn().Msgf("⚠️ GitLab вернул статус %d", resp.StatusCode())
		return nil, ParseGitLabError(resp.Body()) // Используем ParseGitLabError
	}

	var envDetails EnvironmentDetails
	if err := json.Unmarshal(resp.Body(), &envDetails); err != nil {
		log.Error().Err(err).Msg("❌ Ошибка парсинга ответа GitLab")
		return nil, err
	}

	deployment := DeploymentInfo{
		EnvironmentName: envDetails.Name,
		DeploymentDate:  envDetails.LastDeployment.CreatedAt,
		SHA:             envDetails.LastDeployment.SHA,
		Ref:             envDetails.LastDeployment.Ref,
		PipelineID:      envDetails.LastDeployment.Deployable.Pipeline.ID,
		PipelineURL:     envDetails.LastDeployment.Deployable.Pipeline.WebURL,
		JobID:           envDetails.LastDeployment.Deployable.ID,
		JobURL:          envDetails.LastDeployment.Deployable.WebURL,
		DeployStatus:    envDetails.LastDeployment.Deployable.Status,
		BuildCreatedAt:  envDetails.LastDeployment.Deployable.Pipeline.BuildDate,
	}

	// 🔍 Запрашиваем логи джобы, чтобы найти BUILD_VERSION
	buildVersion, err := g.GetBuildVersion(ctx, fmt.Sprintf("%d", deployment.JobID))
	if err != nil {
		log.Warn().Err(err).Msg("⚠️ Не удалось получить BUILD_VERSION, пропускаем")
	} else {
		deployment.BuildVersion = buildVersion
	}

	log.Info().Msgf("✅ Успешно получена информация по окружению %s", envDetails.Name)
	return &deployment, nil
}

// GetBuildVersion - получает BUILD_VERSION из логов джобы
func (g *GitLabClient) GetBuildVersion(ctx context.Context, jobID string) (string, error) {
	if jobID == "" {
		return "", fmt.Errorf("❌ jobID не может быть пустым")
	}

	url := fmt.Sprintf("%s%s%s/jobs/%s/trace", g.baseURL, g.apiURL, g.projectID, jobID)
	log.Debug().Msgf("📡 Запрос логов джобы: jobID=%s, URL=%s", jobID, url)

	resp, err := g.client.R().
		SetContext(ctx).
		Get(url)

	if err != nil {
		log.Error().Err(err).Msg("❌ Ошибка запроса логов GitLab")
		return "", err
	}

	if resp.StatusCode() != http.StatusOK {
		log.Warn().Msgf("⚠️ GitLab вернул статус %d", resp.StatusCode())
		return "", ParseGitLabError(resp.Body()) // Используем ParseGitLabError
	}

	re := regexp.MustCompile(`(?m)^\s*BUILD_VERSION\s*=\s*(\S+)`)
	matches := re.FindStringSubmatch(string(resp.Body()))
	if len(matches) < 2 {
		return "", fmt.Errorf("⚠️ BUILD_VERSION не найден в логах")
	}

	buildVersion := strings.TrimSpace(matches[1])
	log.Info().Msgf("✅ BUILD_VERSION найден: %s", buildVersion)
	return buildVersion, nil
}

// GetPreviousPipelineSHA - ищет SHA предыдущей успешной сборки с пагинацией
func (g *GitLabClient) GetPreviousPipelineSHA(ctx context.Context, ref, currentSHA string) (string, error) {
	if g.projectID == "" {
		return "", fmt.Errorf("❌ projectID не может быть пустым")
	}

	perPage := 100 // Максимальное количество записей на страницу
	page := 1
	foundCurrent := false

	for {
		url := fmt.Sprintf("%s%s%s/pipelines?ref=%s&per_page=%d&page=%d",
			g.baseURL, g.apiURL, g.projectID, ref, perPage, page)
		log.Debug().Msgf("📡 Запрос пайплайнов (страница %d): URL=%s", page, url)

		resp, err := g.client.R().
			SetContext(ctx).
			Get(url)

		if err != nil {
			log.Error().Err(err).Msg("❌ Ошибка запроса пайплайнов GitLab")
			return "", err
		}

		if resp.StatusCode() != http.StatusOK {
			return "", ParseGitLabError(resp.Body())
		}

		var pipelines []Pipeline
		if err := json.Unmarshal(resp.Body(), &pipelines); err != nil {
			log.Error().Err(err).Msg("❌ Ошибка парсинга списка пайплайнов GitLab")
			return "", err
		}

		if len(pipelines) == 0 {
			break // Если больше нет данных, выходим
		}

		for _, pipeline := range pipelines {
			if pipeline.SHA == currentSHA {
				foundCurrent = true
				continue // Пропускаем текущий пайплайн
			}
			if foundCurrent && pipeline.SHA != currentSHA {
				log.Info().Msgf("✅ Найден предыдущий SHA: %s", pipeline.SHA)
				return pipeline.SHA, nil
			}
		}

		page++ // Запрашиваем следующую страницу
	}

	return "", fmt.Errorf("❌ Не удалось найти предыдущий SHA для ref=%s", ref)
}

// GetCommitsBetweenSHAs - получает список коммитов между SHA с поддержкой пагинации
func (g *GitLabClient) GetCommitsBetweenSHAs(ctx context.Context, ref, fromSHA, toSHA string) ([]CommitInfo, error) {
	if g.projectID == "" {
		return nil, fmt.Errorf("❌ projectID не может быть пустым")
	}

	var allCommits []CommitInfo
	foundSHA := false
	perPage := 100 // Максимально возможное значение
	page := 1

	for {
		url := fmt.Sprintf("%s%s%s/repository/commits?ref_name=%s&per_page=%d&page=%d",
			g.baseURL, g.apiURL, g.projectID, ref, perPage, page)
		log.Debug().Msgf("📡 Запрос коммитов (страница %d): URL=%s", page, url)

		resp, err := g.client.R().
			SetContext(ctx).
			Get(url)

		if err != nil {
			log.Error().Err(err).Msg("❌ Ошибка запроса коммитов GitLab")
			return nil, err
		}

		if resp.StatusCode() != http.StatusOK {
			return nil, ParseGitLabError(resp.Body())
		}

		var commits []CommitInfo
		if err := json.Unmarshal(resp.Body(), &commits); err != nil {
			log.Error().Err(err).Msg("❌ Ошибка парсинга коммитов GitLab")
			return nil, err
		}

		if len(commits) == 0 {
			break // Нет больше данных
		}

		for _, commit := range commits {
			if commit.ID == toSHA {
				foundSHA = true
			}
			if commit.ID == fromSHA {
				log.Info().Msgf("✅ Достигнут fromSHA: %s", fromSHA)
				return allCommits, nil
			}
			if foundSHA {
				commit.JiraKeys = ExtractJiraKeys([]CommitInfo{commit}, g.jiraProject)
				allCommits = append(allCommits, commit)
			}
		}

		page++ // Переход на следующую страницу
	}

	if len(allCommits) == 0 {
		return nil, fmt.Errorf("❌ Не найдено новых коммитов между SHA %s и %s", fromSHA, toSHA)
	}

	log.Info().Msgf("✅ Найдено %d новых коммита(ов)", len(allCommits))
	return allCommits, nil
}

// ExtractJiraKeys - ищет Jira-ключи в сообщениях коммитов
func ExtractJiraKeys(commits []CommitInfo, project string) []string {
	jiraRegex := regexp.MustCompile(fmt.Sprintf(`\b%s-\d+\b`, project))
	jiraKeys := make(map[string]bool)

	for _, commit := range commits {
		matches := jiraRegex.FindAllString(commit.Message, -1)
		for _, match := range matches {
			jiraKeys[match] = true
		}
	}

	uniqueKeys := make([]string, 0, len(jiraKeys))
	for key := range jiraKeys {
		uniqueKeys = append(uniqueKeys, key)
	}

	return uniqueKeys
}

// GetPipelineJobs - получает список джоб для указанного pipelineID
func (g *GitLabClient) GetPipelineJobs(ctx context.Context, pipelineID string) ([]JobInfo, error) {
	if pipelineID == "" {
		return nil, fmt.Errorf("❌ pipelineID не может быть пустым")
	}

	url := fmt.Sprintf("%s%s%s/pipelines/%s/jobs", g.baseURL, g.apiURL, g.projectID, pipelineID)
	log.Debug().Msgf("📡 Запрос джоб пайплайна: pipelineID=%s, URL=%s", pipelineID, url)

	resp, err := g.client.R().
		SetContext(ctx).
		Get(url)

	if err != nil {
		log.Error().Err(err).Msg("❌ Ошибка запроса джоб GitLab")
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, ParseGitLabError(resp.Body())
	}

	var jobs []JobInfo
	if err := json.Unmarshal(resp.Body(), &jobs); err != nil {
		log.Error().Err(err).Msg("❌ Ошибка парсинга списка джоб GitLab")
		return nil, err
	}

	// Фильтруем только stage=deploy
	var deployJobs []JobInfo
	for _, job := range jobs {
		if job.Stage == "deploy" {
			deployJobs = append(deployJobs, job)
		}
	}

	log.Info().Msgf("✅ Найдено %d джоб(ы) в deploy-стадии", len(deployJobs))
	return deployJobs, nil
}

// TriggerDeployJob - запускает указанную deploy-джобу
func (g *GitLabClient) TriggerDeployJob(ctx context.Context, jobID string) (*TriggeredJob, error) {
	if jobID == "" {
		return nil, fmt.Errorf("❌ jobID не может быть пустым")
	}

	url := fmt.Sprintf("%s%s%s/jobs/%s/play", g.baseURL, g.apiURL, g.projectID, jobID)
	log.Debug().Msgf("🚀 Запуск деплоя: jobID=%s, URL=%s", jobID, url)

	resp, err := g.client.R().
		SetContext(ctx).
		Post(url)

	if err != nil {
		log.Error().Err(err).Msg("❌ Ошибка запроса на запуск деплоя")
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, ParseGitLabError(resp.Body())
	}

	var triggeredJob TriggeredJob
	if err := json.Unmarshal(resp.Body(), &triggeredJob); err != nil {
		log.Error().Err(err).Msg("❌ Ошибка парсинга ответа GitLab")
		return nil, err
	}

	log.Info().Msgf("✅ Деплой запущен: jobID=%s, статус=%s", jobID, triggeredJob.Status)
	return &triggeredJob, nil
}
