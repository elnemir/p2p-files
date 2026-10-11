SHELL := /bin/sh
export GOMODCACHE := $(CURDIR)/.cache/gomod
export GOCACHE := $(CURDIR)/.cache/go-build
export NEXT_TELEMETRY_DISABLED := 1

.PHONY: check lint test test-race test-integration integration-docker contracts docs-check build up down migrate security e2e ingress
check: lint test-race docs-check contracts build
lint:
	@test -z "$$(gofmt -l cmd internal)" || (gofmt -l cmd internal; exit 1)
	go vet ./...
	npm run typecheck

test:
	go test ./...
	npm run test:contracts
	python3 scripts/test_production_config.py

test-race:
	go test -race ./...
	npm run test:contracts
	python3 scripts/test_production_config.py

test-integration:
	@test -n "$$FILEX_TEST_DATABASE_URL" || (echo 'FILEX_TEST_DATABASE_URL is required; use make integration-docker for an isolated database'; exit 1)
	go test -race -count=1 -run TestIntegration -v ./internal/...

integration-docker:
	python3 scripts/test-integration.py

contracts:
	python3 scripts/build_contracts.py --check
	npm run contracts

docs-check:
	python3 scripts/check_docs.py

build:
	go build ./cmd/...
	npm run build

up:
	docker compose --env-file .env -f deploy/compose.yaml up -d --build

down:
	docker compose --env-file .env -f deploy/compose.yaml down

migrate:
	go run ./cmd/filex-cli migrate

security:
	npm audit --omit=dev --audit-level=high
	go tool govulncheck ./...

e2e:
	python3 scripts/test-e2e.py

ingress:
	python3 scripts/test-ingress.py
