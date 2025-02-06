package service

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/vkr-mtuci/gitlab-service/internal/adapter"
)

// GitLabService - сервис для работы с GitLab API
type GitLabService struct {
	client adapter.GitLabClientInterface // Используем интерфейс для легкого мокирования
}

// NewGitLabService создаёт новый экземпляр GitLabService
func NewGitLabService(client adapter.GitLabClientInterface) *GitLabService {
	return &GitLabService{client: client}
}

// GetEnvironments получает список окружений для проекта
func (s *GitLabService) GetEnvironments() ([]adapter.Environment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	environments, err := s.client.GetEnvironments(ctx)
	if err != nil {
		log.Error().Err(err).Msg("❌ Ошибка получения окружений GitLab")
		return nil, err
	}

	log.Info().Msgf("✅ Успешно получены %d окружений", len(environments))
	return environments, nil
}

// GetEnvironmentDetails получает детальную информацию о конкретном окружении
func (s *GitLabService) GetEnvironmentDetails(environmentID string) (*adapter.DeploymentInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if environmentID == "" {
		log.Warn().Msg("⚠️ Не указан ID окружения")
		return nil, ErrMissingEnvironmentID
	}

	// Запрашиваем информацию о деплое через клиент
	details, err := s.client.GetEnvironmentDetails(ctx, environmentID)
	if err != nil {
		log.Error().Err(err).Msgf("❌ Ошибка получения информации по окружению %s", environmentID)
		return nil, err
	}

	log.Info().Msgf("✅ Успешно получена информация по окружению %s", environmentID)
	return details, nil
}

// ErrMissingEnvironmentID возвращается, если не указан ID окружения
var ErrMissingEnvironmentID = &adapter.GitLabError{
	Message: "ID окружения не указан",
}

// GetCommitsInBuild - получает список коммитов в сборке
func (s *GitLabService) GetCommitsInBuild(ref, currentSHA string) ([]adapter.CommitInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	previousSHA, err := s.client.GetPreviousPipelineSHA(ctx, ref, currentSHA)
	if err != nil {
		log.Warn().Err(err).Msg("⚠️ Не удалось найти предыдущую сборку, возможно первая сборка на этой ветке")
		return nil, err
	}

	commits, err := s.client.GetCommitsBetweenSHAs(ctx, ref, previousSHA, currentSHA)
	if err != nil {
		log.Error().Err(err).Msg("❌ Ошибка получения коммитов между сборками")
		return nil, err
	}

	return commits, nil
}

// GetDeployJobs - получает список джоб в deploy-стадии для указанного пайплайна
func (s *GitLabService) GetDeployJobs(ctx context.Context, pipelineID string) ([]adapter.JobInfo, error) {
	log.Debug().Msgf("📡 Получение deploy-джоб для pipelineID=%s", pipelineID)

	jobs, err := s.client.GetPipelineJobs(ctx, pipelineID)
	if err != nil {
		log.Error().Err(err).Msg("❌ Ошибка получения deploy-джоб")
		return nil, err
	}

	return jobs, nil
}

// TriggerDeployJob - запускает указанную deploy-джобу
func (s *GitLabService) TriggerDeployJob(ctx context.Context, jobID string) (*adapter.TriggeredJob, error) {
	log.Debug().Msgf("🚀 Запуск deploy-джобы jobID=%s", jobID)

	job, err := s.client.TriggerDeployJob(ctx, jobID)
	if err != nil {
		log.Error().Err(err).Msg("❌ Ошибка запуска deploy-джобы")
		return nil, err
	}

	return job, nil
}
