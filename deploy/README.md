# Deploy

## Local compose

```sh
./deploy/quick_start.sh
```

## Scale workers

```sh
SCHEDULERS=2 FETCHERS=4 DELIVERIES=4 ./deploy/scale_workers.sh
```

The scripts read `.env` from the repository root. If `.env` does not exist, they copy `.env.example`.
