.PHONY: tidy download vendor test lint clean help build run

tidy:
	go mod tidy

download:
	go mod download

vendor:
	go mod vendor

test:
	cd plugin && go test -v ./... -coverprofile=coverage.out

lint:
	cd plugin && golangci-lint run ./...

clean:
	docker rmi tix-ttd-gateway || true

help:
	@echo "Available targets:"
	@echo "  tidy         - Run go mod tidy to clean up dependencies"
	@echo "  download     - Download Go modules to local cache"
	@echo "  vendor       - Download modules to vendor directory"
	@echo "  test         - Run Go tests with coverage"
	@echo "  lint         - Run golangci-lint"
	@echo "  check        - Check configuration"
	@echo "  build        - Build production Docker image (requires GITHUB_TOKEN)"
	@echo "  run          - Run the Docker container on port 8888"
	@echo "  clean        - Remove Docker images"
	@echo "  help         - Show this help message"
	@echo ""
	@echo "Usage examples:"
	@echo "  make build GITHUB_TOKEN=your_token"
	@echo "  make run"

check:
	./check_config.sh

build:
	@if [ -z "$(GITHUB_TOKEN)" ]; then \
		echo "Error: GITHUB_TOKEN is required. Usage: make build GITHUB_TOKEN=your_token"; \
		exit 1; \
	fi
	docker build -t platform-krakend --build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) .

run:
	docker run --rm -p "8888:8888" \
    -v ${PWD}/example/config:/etc/krakend \
    -e REDIS_HOST=host.docker.internal:6379 \
    -e REDIS_PASSWORD= \
    platform-krakend
