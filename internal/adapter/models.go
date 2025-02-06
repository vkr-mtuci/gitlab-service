package adapter

import (
	"fmt"
	"time"
)

// GitLabError - структура для обработки ошибок от GitLab API
type GitLabError struct {
	Message string `json:"message"`
}

// Error реализует интерфейс error для GitLabError
func (e *GitLabError) Error() string {
	return fmt.Sprintf("GitLab API Error: %s", e.Message)
}

// Environment - структура для хранения информации об окружении
type Environment struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// EnvironmentDetails - структура с детальной информацией об окружении
type EnvironmentDetails struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	ExternalURL    string `json:"external_url"`
	CreatedAt      string `json:"created_at"`
	LastDeployment struct {
		CreatedAt  string `json:"created_at"`
		Ref        string `json:"ref"`
		SHA        string `json:"sha"`
		Deployable struct {
			ID       int    `json:"id"`
			WebURL   string `json:"web_url"`
			Status   string `json:"status"`
			Pipeline struct {
				ID     int    `json:"id"`
				WebURL string `json:"web_url"`
			} `json:"pipeline"`
		} `json:"deployable"`
	} `json:"last_deployment"`
}

// DeploymentInfo - итоговая структура для хранения информации о деплое
type DeploymentInfo struct {
	EnvironmentName string `json:"environment_name"`
	DeploymentDate  string `json:"deployment_date"`
	Ref             string `json:"ref"`
	SHA             string `json:"sha"`
	PipelineID      int    `json:"pipeline_id"`
	PipelineURL     string `json:"pipeline_url"`
	JobID           int    `json:"job_id"`
	JobURL          string `json:"job_url"`
	DeployStatus    string `json:"deploy_status"`
	BuildVersion    string `json:"build_version"`
}

// Pipeline - структура для хранения информации о пайплайнах
type Pipeline struct {
	ID      int    `json:"id"`
	SHA     string `json:"sha"`
	Ref     string `json:"ref"`
	WebURL  string `json:"web_url"`
	Created string `json:"created_at"`
}

// CommitInfo - структура для хранения информации о коммите
type CommitInfo struct {
	ID          string   `json:"id"`
	CreatedAt   string   `json:"created_at"`
	Message     string   `json:"message"`
	AuthorName  string   `json:"author_name"`
	AuthorEmail string   `json:"author_email"`
	WebURL      string   `json:"web_url"`
	JiraKeys    []string `json:"jira_keys"`
}

// JobInfo - информация о джобе
type JobInfo struct {
	ID         int       `json:"id"`
	Status     string    `json:"status"`
	FinishedAt time.Time `json:"finished_at"`
	Stage      string    `json:"stage"`
	WebURL     string    `json:"web_url"`
	Stand      string    `json:"name"`
}

// TriggeredJob - структура для информации о запущенной джобе
type TriggeredJob struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Stage     string    `json:"stage"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	WebURL    string    `json:"web_url"`
}
