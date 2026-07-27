#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
ENV_FILE="$ROOT_DIR/.env"
ENV_EXAMPLE="$ROOT_DIR/.env.example"
GENERATED_COMPOSE="$ROOT_DIR/deploy/docker-compose.replicas.generated.yml"

if [ ! -f "$ENV_FILE" ]; then
  if [ ! -f "$ENV_EXAMPLE" ]; then
    echo "missing .env and .env.example" >&2
    exit 1
  fi
  cp "$ENV_EXAMPLE" "$ENV_FILE"
fi

set -a
. "$ENV_FILE"
set +a

SCHEDULERS="${SCHEDULERS:-1}"
FETCHERS="${FETCHERS:-1}"
DELIVERIES="${DELIVERIES:-1}"

is_positive_int() {
  case "$1" in
    ''|*[!0-9]*) return 1 ;;
    0) return 1 ;;
    *) return 0 ;;
  esac
}

for value_name in SCHEDULERS FETCHERS DELIVERIES; do
  eval "value=\${$value_name}"
  if ! is_positive_int "$value"; then
    echo "$value_name must be a positive integer, got '$value'" >&2
    exit 1
  fi
done

mkdir -p "$ROOT_DIR/logs/api"

cat > "$GENERATED_COMPOSE" <<'YAML'
services:
YAML

add_worker() {
  service_name="$1"
  cmd_path="$2"
  config_file="$3"
  log_dir="$4"
  depends_on_service="$5"

  mkdir -p "$ROOT_DIR/logs/$log_dir"

  cat >> "$GENERATED_COMPOSE" <<YAML
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
      $depends_on_service:
        condition: service_started
    volumes:
      - ../configs/$config_file:/app/config/core.yml:ro
      - ../configs/logging.yml:/app/config/logging.yml:ro
      - ../logs/$log_dir:/app/logs

YAML
}

i=1
while [ "$i" -le "$SCHEDULERS" ]; do
  add_worker "scheduler-$i" "./cmd/scheduler" "scheduler.yml" "scheduler_$i" "api"
  i=$((i + 1))
done

i=1
while [ "$i" -le "$FETCHERS" ]; do
  add_worker "fetcher-worker-$i" "./cmd/fetcher-worker" "fetcher.yml" "fetcher_$i" "scheduler-1"
  i=$((i + 1))
done

i=1
while [ "$i" -le "$DELIVERIES" ]; do
  add_worker "delivery-worker-$i" "./cmd/delivery-worker" "deliver.yml" "delivery_$i" "scheduler-1"
  i=$((i + 1))
done

docker compose \
  -f "$ROOT_DIR/deploy/docker-compose.yml" \
  -f "$GENERATED_COMPOSE" \
  up --build
