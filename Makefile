.PHONY: run test build docker-build docker-run

run:
	go run cmd/server/main.go

test:
	go test -v -race ./...

build:
	mkdir -p bin
	go build -o bin/server ./cmd/server

docker-build:
	docker build -t sourceasia-backend .

docker-run:
	docker run -p 8080:8080 sourceasia-backend
