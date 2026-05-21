#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
COMPOSE="${COMPOSE:-docker compose}"
REPLICA_COMPOSE="deploy/docker-compose.replicas.generated.yml"

cd "$ROOT_DIR"

if [ ! -f .env ]; then
  cp .env.example .env
fi

set -a
. ./.env
set +a

SCHEDULERS="${SCHEDULERS:-1}"
FETCHERS="${FETCHERS:-1}"
DELIVERIES="${DELIVERIES:-1}"

validate_positive_count() {
  name="$1"
  value="$2"
  case "$value" in
    ''|*[!0-9]*)
      echo "$name must be a positive integer" >&2
      exit 1
      ;;
  esac
  if [ "$value" -lt 1 ]; then
    echo "$name must be greater than zero" >&2
    exit 1
  fi
}

validate_positive_count "SCHEDULERS" "$SCHEDULERS"
validate_positive_count "FETCHERS" "$FETCHERS"
validate_positive_count "DELIVERIES" "$DELIVERIES"

mkdir -p logs/api

cat > "$REPLICA_COMPOSE" <<EOF
services:
EOF

append_worker() {
  service_name="$1"
  cmd_path="$2"
  config_path="$3"
  log_path="$4"
  depends_on_scheduler="$5"

  cat >> "$REPLICA_COMPOSE" <<EOF
  $service_name:
    env_file:
      - ../.env
    build:
      context: ..
      dockerfile: ./deploy/Dockerfile
      args:
        CMD_PATH: $cmd_path
    depends_on:
      postgres-migrate:
        condition: service_completed_successfully
      nats-init:
        condition: service_completed_successfully
      otel-collector:
        condition: service_started
EOF

  if [ "$depends_on_scheduler" = "true" ]; then
    cat >> "$REPLICA_COMPOSE" <<EOF
      scheduler-1:
        condition: service_started
EOF
  else
    cat >> "$REPLICA_COMPOSE" <<EOF
      api:
        condition: service_started
EOF
  fi

  cat >> "$REPLICA_COMPOSE" <<EOF
    volumes:
      - ../$config_path:/app/config/core.yml:ro
      - ../configs/logging.yml:/app/config/logging.yml:ro
      - ../$log_path:/app/logs

EOF
}

i=1
while [ "$i" -le "$SCHEDULERS" ]; do
  mkdir -p "logs/scheduler_$i"
  append_worker "scheduler-$i" "./cmd/scheduler" "configs/scheduler.yml" "logs/scheduler_$i" "false"
  i=$((i + 1))
done

i=1
while [ "$i" -le "$FETCHERS" ]; do
  mkdir -p "logs/fetcher_$i"
  append_worker "fetcher-worker-$i" "./cmd/fetcher-worker" "configs/fetcher.yml" "logs/fetcher_$i" "true"
  i=$((i + 1))
done

i=1
while [ "$i" -le "$DELIVERIES" ]; do
  mkdir -p "logs/delivery_$i"
  append_worker "delivery-worker-$i" "./cmd/delivery-worker" "configs/deliver.yml" "logs/delivery_$i" "true"
  i=$((i + 1))
done

$COMPOSE --env-file .env -f deploy/docker-compose.yml -f "$REPLICA_COMPOSE" up -d --build \
  postgres \
  nats \
  postgres-migrate \
  nats-init \
  jaeger \
  otel-collector \
  prometheus \
  api

$COMPOSE --env-file .env -f deploy/docker-compose.yml -f "$REPLICA_COMPOSE" up -d --build

$COMPOSE --env-file .env -f deploy/docker-compose.yml -f "$REPLICA_COMPOSE" ps
