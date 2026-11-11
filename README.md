# SourceAsia Product API

A small Go API assignment demonstrating request throttling and a product catalog with paginated summaries and product media. It runs without external services and stores data in memory.

## Run locally

Requirements: Go 1.21 or newer.

```sh
go mod download
go run ./cmd/server
```

The server listens on `http://localhost:8080`. Set `PORT` to use a different port.

## API

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/health` | Liveness check |
| `POST` | `/request` | Submit a rate-limited request |
| `GET` | `/stats` | View accepted and rejected counts by supplied user ID |
| `POST` | `/products` | Create a product |
| `GET` | `/products?limit=20&offset=0` | List product summaries |
| `GET` | `/products/{id}` | Read product details |
| `POST` | `/products/{id}/media` | Append image and video URLs |

Example product request:

```sh
curl -X POST http://localhost:8080/products \
  -H 'Content-Type: application/json' \
  -d '{"name":"Desk lamp","sku":"LAMP-001","image_urls":["https://example.com/lamp.jpg"]}'
```

`/request` expects `{"user_id":"demo-user","payload":{}}`. It accepts up to five requests per user ID per rolling minute. Product list limits default to 20 and cap at 100; each media request accepts at most 20 URLs per type. JSON request bodies are limited to 1 MiB.

## Checks

```sh
go test ./...
go vet ./...
go build ./cmd/server
```

The GitHub Actions workflow runs formatting, vet, tests with the race detector, and a build on pushes and pull requests.

## Run with Docker

```sh
docker build -t sourceasia-product-api .
docker run --rm -p 8080:8080 sourceasia-product-api
```

## Limits before internet deployment

This assignment has no authentication, database, or distributed rate limiter. User IDs are supplied by the caller, `/stats` exposes those IDs and counts, and all data resets when the process restarts. Use it for local evaluation or a protected demo only; add authentication, access controls, persistent storage, and a trusted rate-limit identity before exposing it as a production API.
