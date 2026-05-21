#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
COMPOSE="${COMPOSE:-docker compose}"
SCHEDULERS="${SCHEDULERS:-1}"
FETCHERS="${FETCHERS:-1}"
DELIVERIES="${DELIVERIES:-1}"

cd "$ROOT_DIR"

if [ ! -f .env ]; then
  cp .env.example .env
fi

mkdir -p logs/api logs/scheduler logs/fetcher logs/delivery

$COMPOSE --env-file .env -f deploy/docker-compose.yml up -d --build \
  --scale scheduler="$SCHEDULERS" \
  --scale fetcher-worker="$FETCHERS" \
  --scale delivery-worker="$DELIVERIES" \
  scheduler \
  fetcher-worker \
  delivery-worker

$COMPOSE --env-file .env -f deploy/docker-compose.yml ps
