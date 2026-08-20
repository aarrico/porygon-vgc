# Single entrypoint for every developer-facing command. Later stories add
# targets here (1.1b: migrate, 1.1c: etl) rather than a second entrypoint.

COMPOSE_FILE := deploy/compose/docker-compose.yml
COMPOSE      := docker compose -f $(COMPOSE_FILE)
API_URL      ?= http://localhost:8080
WAIT_TIMEOUT ?= 120

# Bare `make` must not touch Docker.
.DEFAULT_GOAL := help
# verify's prerequisites are ordered; -j must not interleave them.
.NOTPARALLEL:

.PHONY: help up down clean logs build vet test lint fmt tidy health verify

## help: list the available targets
help:
	@grep -hE '^## [a-z-]+:' $(MAKEFILE_LIST) | sed 's/^## //' | awk -F': ' '{printf "  %-8s %s\n", $$1, $$2}'

## up: build images and bring the stack up, blocking until both are healthy
up:
	$(COMPOSE) up -d --build --wait --wait-timeout $(WAIT_TIMEOUT)

## down: tear the stack down; the named volume is retained
down:
	$(COMPOSE) down

## clean: tear the stack down and drop the named volume and any orphans
clean:
	$(COMPOSE) down --volumes --remove-orphans

## logs: follow logs for every service
logs:
	$(COMPOSE) logs -f

## build: compile every Go package
build:
	cd backend && go build ./...

## vet: run go vet over the module
vet:
	cd backend && go vet ./...

## test: run the Go test suite
test:
	cd backend && go test ./...

## lint: run golangci-lint over the module
lint:
	cd backend && golangci-lint run

## fmt: rewrite every Go file with gofmt
fmt:
	cd backend && gofmt -w .

## tidy: reconcile go.mod and go.sum with the imports
tidy:
	cd backend && go mod tidy

## health: print the /healthz body and status, whatever the status is
health:
	@curl -s -w '\nHTTP %{http_code}\n' $(API_URL)/healthz \
		|| echo "curl failed (exit $$?) — no listener at $(API_URL); try: make up"

## verify: gofmt check + build + vet + test + lint
verify: build vet test lint
	@cd backend && unformatted=$$(gofmt -l .); \
		if [ -n "$$unformatted" ]; then \
			echo "gofmt needed on:"; echo "$$unformatted"; exit 1; \
		fi
