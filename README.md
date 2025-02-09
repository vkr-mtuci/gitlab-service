# GitLab Service

## 📌 Описание
**GitLab Service** – это REST API-сервис, предназначенный для взаимодействия с GitLab API. Он позволяет получать информацию об окружениях, пайплайнах, коммитах и запускать деплой-джобы.

## 🚀 Функциональность
- Получение списка окружений проекта
- Получение деталей конкретного окружения
- Получение списка коммитов между сборками
- Получение списка deploy-джоб для пайплайна
- Запуск deploy-джобы

## 🛠 Технологии
- **Go (Golang)** – основной язык разработки
- **Fiber** – веб-фреймворк для обработки HTTP-запросов
- **Zerolog** – логирование
- **Resty** – клиент для HTTP-запросов
- **GitLab API** – взаимодействие с GitLab

## 📦 Установка и настройка
### 1️⃣ Клонирование репозитория
```sh
git clone https://github.com/vkr-mtuci/gitlab-service.git
cd gitlab-service
```

### 2️⃣ Установка зависимостей
```sh
go mod tidy
```

### 3️⃣ Создание файла конфигурации `.env`
```sh
touch .env
```
Добавьте в `.env` необходимые переменные окружения:
```
SERVER_PORT=8080
GITLAB_BASE_URL=https://gitlab.com
GITLAB_API_URL=/api/v4/projects/
GITLAB_API_TOKEN=your_personal_access_token
GITLAB_PROJECT_ID=your_project_id
JIRA_PROJECT=your_jira_project
```

### 4️⃣ Запуск сервиса
```sh
go run cmd/main.go
```

## 📡 API-эндпоинты
### 📌 Получение списка окружений
**GET /environments**
```json
{
  "environments": [
    { "id": 1, "name": "staging" },
    { "id": 2, "name": "production" }
  ]
}
```

### 📌 Получение деталей окружения
**GET /environments/:id**
```json
{
  "environment_name": "staging",
  "deployment_date": "2025-02-06T21:22:14Z",
  "ref":"develop",
  "deploy_status": "success",
  "sha":"b0f9951803dcd80c141b667acbca9e46bace8acf",
  "pipeline_id":11111111,
  "pipeline_url":"https://gitlab.example.ru/group/project/-/pipelines/11111111",
  "job_id":2222222,
  "job_url":"https:/gitlab.example.ru/group/project/-/jobs/2222222",
  "build_version":"1.1.0"
}
```

### 📌 Получение коммитов между сборками
**GET /commits/:ref/:sha**
```json
{
  "commits": [
    { "id":"","created_at":"","message":"","author_name":"","author_email":"","web_url":"","jira_keys":[] }
  ]
}
```

### 📌 Получение deploy-джоб
**GET /pipelines/:pipeline_id/deploy-jobs**
```json
{
  "deploy_jobs": [
    { "id": 7, "status": "success", "finished_at":"0001-01-01T00:00:00Z", "stage":"deploy", "web_url": "https://gitlab.com/job/7", "name":"" }
  ]
}
```

### 📌 Запуск deploy-джобы
**POST /jobs/:job_id/play**
```json
{
  "id": 7,
  "status": "pending",
  "web_url": "https://gitlab.com/job/7"
}
```

## ✅ Тестирование
Запуск тестов с покрытием кода:
```sh
go test ./... -cover
```
Проверка покрытия кода:
```sh
go tool cover -func=./coverage/coverage.out
```

