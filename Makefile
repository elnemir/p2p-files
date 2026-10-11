SHELL := /bin/sh
export GOMODCACHE := $(CURDIR)/.cache/gomod
export GOCACHE := $(CURDIR)/.cache/go-build
export NEXT_TELEMETRY_DISABLED := 1
export STATICCHECK_CACHE := $(CURDIR)/.cache/staticcheck

.PHONY: check lint test test-race test-integration integration-docker contracts docs-check build up down migrate security e2e ingress upgrade-drill fuzz performance supply-chain release-check
check: lint test-race docs-check repository-check contracts build

.PHONY: repository-check
repository-check:
	node scripts/check_repository.mjs
	python3 -m unittest discover -s scripts -p test_github_repository.py
lint:
	@test -z "$$(gofmt -l cmd internal)" || (gofmt -l cmd internal; exit 1)
	go vet ./...
	go tool staticcheck ./...
	npm run typecheck
	npm run lint
	npm run format:check

test:
	go test ./...
	npm run test:contracts
	python3 scripts/test_production_config.py
	python3 scripts/test_contract_compatibility.py
	python3 scripts/test_acceptance.py

test-race:
	go test -race ./...
	npm run test:contracts
	python3 scripts/test_production_config.py
	python3 scripts/test_contract_compatibility.py
	python3 scripts/test_acceptance.py

test-integration:
	@test -n "$$FILEX_TEST_DATABASE_URL" || (echo 'FILEX_TEST_DATABASE_URL is required; use make integration-docker for an isolated database'; exit 1)
	go test -race -count=1 -run TestIntegration -v ./internal/...

integration-docker:
	python3 scripts/test-integration.py

contracts:
	python3 scripts/build_contracts.py --check
	npm run contracts
	npm run check:types

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
	npm audit --audit-level=high
	go tool govulncheck ./...

e2e:
	python3 scripts/test-e2e.py

ingress:
	python3 scripts/test-ingress.py

upgrade-drill:
	python3 scripts/test-upgrade.py

fuzz:
	python3 scripts/test-fuzz.py

performance:
	python3 scripts/test-performance.py

supply-chain:
	python3 scripts/supply_chain.py source
	python3 scripts/supply_chain.py licenses
	python3 scripts/supply_chain.py images

release-check:
	python3 scripts/acceptance.py readiness
