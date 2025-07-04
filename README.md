# GURLS-Redirect

HTTP сервис для обработки редиректов коротких ссылок GURLS.

## Описание

GURLS-Redirect обрабатывает HTTP запросы на короткие ссылки, получает информацию о ссылках от Backend через gRPC и выполняет редиректы. Также записывает аналитику кликов.

## Функциональность

- Обработка редиректов по коротким ссылкам
- Проверка истечения срока действия ссылок
- Аналитика кликов с определением типа устройства
- Health check endpoints
- Swagger UI для API документации

## Запуск

### Локальная разработка

```bash
# Установка зависимостей
go mod tidy

# Запуск сервера
go run ./cmd/redirect

# Или сборка и запуск
go build -o bin/redirect ./cmd/redirect
./bin/redirect
```

### Конфигурация

Сервис использует файл `config/local.yml` или переменные окружения:

- `HTTP_SERVER_ADDRESS` - адрес HTTP сервера (по умолчанию: 0.0.0.0:8080)
- `BASE_URL` - базовый URL для коротких ссылок
- `GRPC_BACKEND_ADDRESS` - адрес gRPC Backend сервиса (по умолчанию: localhost:50051)
- `ENV` - окружение (local/dev/production)

### Docker

```bash
# Сборка
docker build -t gurls-redirect .

# Запуск
docker run -p 8080:8080 gurls-redirect
```

## API Endpoints

- `GET /{alias}` - Редирект по короткой ссылке
- `GET /health` - Health check
- `GET /ready` - Readiness probe
- `GET /api/v1/` - Swagger UI

## Архитектура

- `cmd/redirect/` - точка входа приложения
- `internal/handler/http/` - HTTP обработчики
- `internal/grpc/client/` - gRPC клиент для Backend
- `internal/config/` - конфигурация
- `assets/` - ресурсы (regexes.yaml для UA parsing)

## Зависимости

- GURLS-Backend должен быть запущен и доступен
- Файл `assets/regexes.yaml` для парсинга User-Agent

### Загрузка regexes.yaml

```bash
curl -o assets/regexes.yaml https://raw.githubusercontent.com/ua-parser/uap-core/master/regexes.yaml
```