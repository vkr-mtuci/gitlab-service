package test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vkr-mtuci/gitlab-service/internal/adapter"
	"github.com/vkr-mtuci/gitlab-service/internal/service"
	"github.com/vkr-mtuci/gitlab-service/test/mocks"
)

func TestService_TriggerDeployJob_Success(t *testing.T) {
	mockClient := new(mocks.MockGitLabClient)
	service := service.NewGitLabService(mockClient)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Мокируем успешный запуск
	mockClient.On("TriggerDeployJob", ctx, "7").Return(&adapter.TriggeredJob{
		ID:        7,
		Name:      "deploy-production",
		Stage:     "deploy",
		Status:    "running",
		CreatedAt: time.Now(),
		WebURL:    "https://example.com/foo/bar/-/jobs/7",
	}, nil)

	// Вызываем сервис
	job, err := service.TriggerDeployJob(ctx, "7")

	// Проверяем результат
	require.NoError(t, err)
	assert.NotNil(t, job)
	assert.Equal(t, 7, job.ID)
	assert.Equal(t, "pending", job.Status)
	assert.Equal(t, "deploy-production", job.Name)
	assert.Equal(t, "deploy", job.Stage)
	assert.Equal(t, "https://example.com/foo/bar/-/jobs/7", job.WebURL)
}

func TestServiceGetEnvironments_Success(t *testing.T) {
	mockClient := &mocks.MockGitLabClient{}
	svc := service.NewGitLabService(mockClient)

	environments, err := svc.GetEnvironments()

	assert.NoError(t, err)
	assert.Len(t, environments, 2)
	assert.Equal(t, "staging", environments[0].Name)
	assert.Equal(t, "production", environments[1].Name)
}

func TestServiceGetEnvironmentDetails_Success(t *testing.T) {
	mockClient := &mocks.MockGitLabClient{}
	svc := service.NewGitLabService(mockClient)

	environment, err := svc.GetEnvironmentDetails("1")

	assert.NoError(t, err)
	assert.NotNil(t, environment)
	assert.Equal(t, "staging", environment.EnvironmentName)
	assert.Equal(t, "success", environment.DeployStatus)
	assert.Equal(t, "1.2.3", environment.BuildVersion)
}

func TestGetCommitsInBuild_Success(t *testing.T) {
	mockClient := &mocks.MockGitLabClient{}
	svc := service.NewGitLabService(mockClient)

	commits, err := svc.GetCommitsInBuild("develop", "sha-123")

	assert.NoError(t, err)
	assert.Len(t, commits, 2)
	assert.Equal(t, "commit-1", commits[0].ID)
	assert.Equal(t, "JIRA-123", commits[0].JiraKeys[0])

	// ✅ Проверяем второй коммит
	assert.Equal(t, "commit-2", commits[1].ID)
	assert.Equal(t, "JIRA-456", commits[1].JiraKeys[0])
}

func TestGetCommitsInBuild_NoCommits(t *testing.T) {
	mockClient := &mocks.MockGitLabClient{}
	svc := service.NewGitLabService(mockClient)

	commits, err := svc.GetCommitsInBuild("develop", "unknown-sha")

	assert.Error(t, err)
	assert.Nil(t, commits)
}
