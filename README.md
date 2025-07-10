# FastNetMon Blackhole Prometheus Exporter

A Prometheus exporter written in Go that collects blocked IP addresses from the FastNetMon API.

## Features

- Pulls blocked IPs from FastNetMon's REST API (`/blackhole` endpoint).
- Exposes them as Prometheus metrics: `blocked_ip{ip="..."}`.
- Proper error and authorization checks with readable logging.
- All configuration via environment variables (`.env`-ready).
- Production-ready for Docker and CI/CD deployments.
- Healthcheck endpoint at `/health`.

---

## Environment Variables

| Variable             | Description                                  | Example                                   |
|----------------------|----------------------------------------------|-------------------------------------------|
| `EXPORTER_API_URL`   | FastNetMon API URL (blackhole endpoint)      | `http://example.com/blackhole`            |
| `EXPORTER_USER`      | API username                                 | `api`                                     |
| `EXPORTER_PASSWORD`  | API password                                 | `password`                                |
| `EXPORTER_PORT`      | Port to run the exporter on                  | `:9898`                                   |

---

## Quick Start (Locally)

1. Create a `.env` file in your project root:
    ```env
    EXPORTER_API_URL=http://example.com/blackhole
    EXPORTER_USER=api
    EXPORTER_PASSWORD=123
    EXPORTER_PORT=:9898
    ```
2. Install dependencies and run the exporter:
    ```sh
    go mod tidy
    go run main.go
    ```

---

## Quick Start with Docker

1. Build the image:
    ```sh
    docker build -t fastnetmon_exporter .
    ```
2. Run using your `.env` file:
    ```sh
    docker run --rm -p 9898:9898 --env-file .env fastnetmon_exporter
    ```
3. Check Prometheus metrics:
    ```sh
    curl http://localhost:9898/metrics
    ```
4. Check health endpoint:
    ```sh
    curl http://localhost:9898/health
    ```

---
