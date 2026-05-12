#!/usr/bin/env sh
set -eu

COMPOSE="${COMPOSE:-docker compose}"

mkdir -p logs/api logs/scheduler logs/fetcher logs/delivery

$COMPOSE up -d --build \
  postgres \
  nats \
  postgres-migrate \
  nats-init \
  api \
  scheduler \
  fetcher-worker \
  delivery-worker

$COMPOSE ps
