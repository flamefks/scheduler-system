# Scheduler System

Scheduler System is a Go-based job scheduling pipeline built around PostgreSQL, NATS JetStream, HTTP workers, and OpenTelemetry metrics.

The project is currently in beta. It is usable for local experiments and active development, but it is not finished yet and should not be treated as production-ready without additional hardening.

## What It Does

- Stores jobs, schedules, fetcher configuration, and delivery configuration in PostgreSQL.
- Uses the API service to create, patch, activate, deactivate, and inspect jobs.
- Uses the scheduler service to claim due jobs and publish work into NATS JetStream.
- Uses fetcher workers to execute source HTTP requests and publish results forward.
- Uses delivery workers to send fetched payloads to target HTTP endpoints.
- Exposes OpenTelemetry metrics through an OTEL Collector and Prometheus.
- Includes Jaeger wiring for tracing infrastructure, while tracing usage is still being expanded.

## Services

- `api`: HTTP API for job management.
- `scheduler`: claims due jobs and pushes job IDs to NATS.
- `fetcher-worker`: consumes fetch jobs, calls configured source endpoints, and publishes delivery jobs.
- `delivery-worker`: consumes delivery jobs and calls configured target endpoints.
- `postgres-migrate`: applies database migrations.
- `nats-init`: creates or updates required JetStream streams and consumers.

## Current Status

This is still a beta project.

Known areas that still need work:

- NATS retry, ack, nak, term, and final-failure behavior still needs a stricter design pass.
- Worker concurrency and consumer configuration are still evolving.
- File logging works for local deploys, but production log shipping should be planned deliberately.
- Kubernetes or Helm deployment files are not finalized.
- Tracing is initialized, but spans are not yet consistently added through business flows.

## Quick Start

Copy the environment example and edit it if needed:

```sh
cp .env.example .env
```

Start the local stack:

```sh
./deploy/quick_start.sh
```

The quick start script reads replica counts from `.env`:

```env
SCHEDULERS=1
FETCHERS=1
DELIVERIES=1
```

For each worker replica, the script generates an isolated log directory, for example:

```text
logs/scheduler_1/
logs/fetcher_1/
logs/delivery_1/
```

## Observability

- Prometheus: http://localhost:9090
- Jaeger UI: http://localhost:16686
- OTEL gRPC endpoint inside Docker: `otel-collector:4317`

Metrics are exported from services through OpenTelemetry to the collector. Prometheus scrapes the collector.

## Configuration

Default config files live in `configs/`.

Environment values are loaded from `.env`, then expanded inside YAML configs. See `.env.example` for the expected variables.

## Development

Run tests:

```sh
go test ./... -count=1
```

Regenerate sqlc code after SQL changes:

```sh
sqlc generate
```

## Documentation

Russian README: [README.ru.md](README.ru.md)
