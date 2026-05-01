# ---------- BUILD STAGE ----------
FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG CMD_PATH

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /bin/app ${CMD_PATH}

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

RUN adduser -D appuser
USER appuser

COPY --from=builder /bin/app /app/app
COPY config /app/config

ENTRYPOINT ["/app/app"]
