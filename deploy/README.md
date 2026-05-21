# Deploy

## Local compose

```sh
./deploy/quick_start.sh
```

Replica counts are read from the repository `.env` file:

```env
SCHEDULERS=1
FETCHERS=1
DELIVERIES=1
```

`quick_start.sh` generates `deploy/docker-compose.replicas.generated.yml` and gives each worker instance its own host log directory:

```text
logs/scheduler_1/
logs/fetcher_1/
logs/delivery_1/
```

Prometheus UI: http://localhost:9090

Jaeger UI: http://localhost:16686

The script reads `.env` from the repository root. If `.env` does not exist, it copies `.env.example`.
