.PHONY: build build-daemon build-cli run-daemon run-cli test tidy vet install ensure-tools migrate-new migrate-up migrate-down migrate-fresh docker-build docker-run docker-test docker-up docker-down docker-cli-compose

# Auto-install required CLI tools
ensure-tools:
	@which swag > /dev/null 2>&1 || go install github.com/swaggo/swag/cmd/swag@latest
	@which migrate > /dev/null 2>&1 || go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

DAEMON_BINARY = bin/homelytics-daemon
CLI_BINARY    = bin/homelytics-agent

# Build both binaries
build: build-daemon build-cli

# Build the daemon binary
build-daemon:
	go build -o $(DAEMON_BINARY) ./cmd/daemon

# Build the CLI binary
build-cli:
	go build -o $(CLI_BINARY) ./cmd/cli

# Run the daemon with the default config
run-daemon:
	go run ./cmd/daemon --config files/config/app.yaml

# Run the CLI (pass args, e.g. make run-cli ARGS="status")
run-cli:
	go run ./cmd/cli $(ARGS)

# Run all tests
test:
	go test ./...

# Tidy module dependencies
tidy:
	go mod tidy

# Run static analysis
vet:
	go vet ./...

# Run the installer
install:
	./install.sh

# Build the Docker image
docker-build:
	docker compose build --no-cache

# Start the daemon container via docker-compose
docker-up:
	mkdir -p var/run
	chmod 777 var/run
	docker compose up -d daemon

# Stop the daemon container
docker-down:
	docker compose down

# Run a one-off CLI command against the running daemon via docker-compose
docker-cli-compose:
	docker compose run --rm --entrypoint "" cli \
		homelytics-agent --config /opt/homelytics/etc/config.yaml $(ARGS)

# Build image and run a quick login/status test in a single container
docker-test:
	make docker-build
	make docker-up
	sleep 2
	docker compose run --rm --entrypoint "" cli \
		homelytics-agent --config /opt/homelytics/etc/config.yaml login --email merchant@example.com --password password
	docker compose run --rm --entrypoint "" cli \
		homelytics-agent --config /opt/homelytics/etc/config.yaml tsnet auth
	docker compose run --rm --entrypoint "" cli \
		homelytics-agent --config /opt/homelytics/etc/config.yaml status

# Create a new migration: make migrate-new name=create_users_table
migrate-new:
	@which migrate > /dev/null 2>&1 || go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	migrate create -ext sql -dir files/migrations -seq $(name)

# Run all pending migrations
migrate-up:
	go run ./cmd/migrate up

# Roll back the last migration
migrate-down:
	go run ./cmd/migrate down

# Drop all tables then re-run all migrations
migrate-fresh:
	go run ./cmd/migrate fresh
