package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"github.com/vkr-mtuci/gitlab-service/internal/service"
)

// GitLabHandler - обработчик запросов для GitLab
type GitLabHandler struct {
	service *service.GitLabService
}

// NewGitLabHandler создаёт новый обработчик
func NewGitLabHandler(service *service.GitLabService) *GitLabHandler {
	return &GitLabHandler{service: service}
}

// GetEnvironments обрабатывает запрос списка окружений
func (h *GitLabHandler) GetEnvironments(c *fiber.Ctx) error {
	environments, err := h.service.GetEnvironments()
	if err != nil {
		log.Error().Err(err).Msg("❌ Ошибка получения окружений GitLab")
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Ошибка при получении окружений",
		})
	}
	return c.JSON(fiber.Map{"environments": environments})
}

// GetEnvironmentDetails обрабатывает запрос детальной информации об окружении
func (h *GitLabHandler) GetEnvironmentDetails(c *fiber.Ctx) error {
	environmentID := c.Params("id")
	if environmentID == "" {
		log.Warn().Msg("⚠️ Не указан ID окружения")
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Необходимо указать environment_id",
		})
	}

	envDetails, err := h.service.GetEnvironmentDetails(environmentID)
	if err != nil {
		log.Error().Err(err).Msgf("❌ Ошибка получения деталей окружения %s", environmentID)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Ошибка при получении данных окружения",
		})
	}

	return c.JSON(envDetails)
}

// GetCommitsInBuild обрабатывает запрос на получение списка коммитов в сборке
func (h *GitLabHandler) GetCommitsInBuild(c *fiber.Ctx) error {
	ref := c.Params("ref")
	sha := c.Params("sha")

	if ref == "" || sha == "" {
		log.Warn().Msg("⚠️ Не указаны ref и sha")
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Необходимо указать ref и sha",
		})
	}

	commits, err := h.service.GetCommitsInBuild(ref, sha)
	if err != nil {
		log.Error().Err(err).Msgf("❌ Ошибка получения коммитов для сборки %s", sha)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Ошибка при получении коммитов",
		})
	}

	return c.JSON(fiber.Map{"commits": commits})
}

// GetDeployJobs обрабатывает запрос на получение списка deploy-джоб
func (h *GitLabHandler) GetDeployJobs(c *fiber.Ctx) error {
	pipelineID := c.Params("pipeline_id")

	if pipelineID == "" {
		log.Warn().Msg("⚠️ Не указан pipeline_id")
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Необходимо указать pipeline_id",
		})
	}

	// Создаём контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	deployJobs, err := h.service.GetDeployJobs(ctx, pipelineID)
	if err != nil {
		log.Error().Err(err).Msgf("❌ Ошибка получения deploy-джоб для pipelineID=%s", pipelineID)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Ошибка при получении deploy-джоб",
		})
	}

	return c.JSON(fiber.Map{"deploy_jobs": deployJobs})
}

// TriggerDeployJob запускает deploy-джобу
func (h *GitLabHandler) TriggerDeployJob(c *fiber.Ctx) error {
	jobID := c.Params("job_id")

	if jobID == "" {
		log.Warn().Msg("⚠️ Не указан job_id")
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Необходимо указать job_id",
		})
	}

	// Создаём контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	jobInfo, err := h.service.TriggerDeployJob(ctx, jobID)
	if err != nil {
		log.Error().Err(err).Msgf("❌ Ошибка запуска deploy-джобы jobID=%s", jobID)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Ошибка при запуске deploy-джобы",
		})
	}

	return c.JSON(jobInfo)
}
