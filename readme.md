# SourceAsia Backend Assignment

A production-quality Go backend service featuring a thread-safe sliding window rate limiter and a performance-optimized product catalog with in-memory storage.

## Architecture

The project follows **Clean Architecture** principles, separating concerns into distinct layers:

- **cmd/server**: Application entry point and dependency injection.
- **internal/models**: Domain entities and DTOs.
- **internal/store**: In-memory repository implementations using mutexes for concurrency safety.
- **internal/services**: Business logic (Rate Limiting, Product Management).
- **internal/handlers**: HTTP transport layer.
- **internal/middleware**: Cross-cutting concerns (Logging, Recovery).
- **internal/validators**: Request validation logic.

### Key Design Decisions

1. **Sliding Window Rate Limiter**: Implemented using a per-user slice of timestamps to ensure accurate rolling 1-minute windows.
2. **List vs. Detail Optimization**: The `Product` model is split internally. The store maintains a fast-access list of "Summaries" to ensure `GET /products` remains $O(limit)$ regardless of total media counts, avoiding expensive serialization of large URL arrays.
3. **Concurrency**: All in-memory maps are protected by `sync.RWMutex` to ensure thread safety under parallel load.

## API Documentation

### Rate Limiter
- `POST /request`: Submit a payload. (Max 5/min per user).
- `GET /stats`: View per-user acceptance and rejection counts.

### Product Catalog
- `POST /products`: Create a product with SKU validation.
- `GET /products`: List products (paginated, optimized).
- `GET /products/{id}`: Get full product details including all media.
- `POST /products/{id}/media`: Append media to an existing product.

### System
- `GET /health`: Service health check.

## Getting Started

### Prerequisites
- Go 1.21+
- Docker (optional)
- Make (optional)

### Running Locally
```bash
# Using Makefile
make run

# Using Go directly
go run cmd/server/main.go
```

### Running with Docker
```bash
docker build -t sourceasia-backend .
docker run -p 8080:8080 sourceasia-backend
```

### Testing
```bash
# Run all tests (including concurrency tests)
go test -v ./...
```

## Production Considerations
In a real-world scenario:
- **Storage**: PostgreSQL or MongoDB would replace the in-memory store.
- **Rate Limiting**: Redis-based sliding windows (using Sorted Sets) would be used for distributed rate limiting.
- **Media**: URLs would point to a CDN (e.g., CloudFront/Cloudinary).
- **Observability**: Prometheus metrics and structured JSON logging (e.g., Zap/Slog) would be integrated.

---
*Scaffolded with AI assistance.*
