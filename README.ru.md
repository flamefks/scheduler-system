# Scheduler System

Scheduler System - это Go-проект для планирования и выполнения задач через PostgreSQL, NATS JetStream, HTTP-воркеры и OpenTelemetry-метрики.

Проект сейчас в beta-состоянии. Он подходит для локальной разработки и экспериментов, но еще не закончен и не должен считаться production-ready без дополнительной доработки.

## Что Делает Проект

- Хранит jobs, schedules, fetcher config и delivery config в PostgreSQL.
- Дает API для создания, patch, activate, deactivate и просмотра jobs.
- Scheduler забирает задачи, которым пора выполняться, и публикует work items в NATS JetStream.
- Fetcher workers выполняют source HTTP-запросы и публикуют результат дальше.
- Delivery workers доставляют payload в target HTTP endpoints.
- Метрики идут через OpenTelemetry Collector в Prometheus.
- Jaeger-инфра подключена, но полноценная трассировка бизнес-флоу еще в работе.

## Сервисы

- `api`: HTTP API для управления jobs.
- `scheduler`: выбирает готовые jobs и отправляет их в NATS.
- `fetcher-worker`: читает fetch jobs, делает HTTP-запросы и публикует delivery jobs.
- `delivery-worker`: читает delivery jobs и делает HTTP-запросы доставки.
- `postgres-migrate`: применяет миграции базы.
- `nats-init`: создает или обновляет нужные JetStream streams и consumers.

## Текущее Состояние

Проект все еще beta.

Что еще нужно доделать:

- Пересобрать и зафиксировать дизайн NATS retry, ack, nak, term и final-failure поведения.
- Довести конфигурацию concurrency и consumers.
- Продумать production-логирование и shipping логов.
- Подготовить Kubernetes или Helm deploy.
- Добавить полноценные spans по бизнес-флоу, сейчас tracing только инициализируется.

## Быстрый Старт

Создай `.env` из примера:

```sh
cp .env.example .env
```

Подними локальный stack:

```sh
./deploy/quick_start.sh
```

Скрипт читает количество экземпляров из `.env`:

```env
SCHEDULERS=1
FETCHERS=1
DELIVERIES=1
```

Для каждого экземпляра воркера создается отдельная директория логов:

```text
logs/scheduler_1/
logs/fetcher_1/
logs/delivery_1/
```

## Observability

- Prometheus: http://localhost:9090
- Jaeger UI: http://localhost:16686
- OTEL gRPC endpoint внутри Docker: `otel-collector:4317`

Метрики сервисов отправляются через OpenTelemetry в collector. Prometheus забирает метрики из collector.

## Конфигурация

Дефолтные конфиги лежат в `configs/`.

Переменные окружения берутся из `.env` и подставляются в YAML-конфиги. Список переменных смотри в `.env.example`.

## Разработка

Запуск тестов:

```sh
go test ./... -count=1
```

После изменения SQL-запросов:

```sh
sqlc generate
```

## Документация

Основной README на английском: [README.md](README.md)
