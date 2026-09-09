COMPOSE := docker compose
GO := go

VERSION ?= $(shell date -u "+%y %m %d %H %M" | awk '{printf "%02d.%02d.%05d", $$1, $$2, (($$3 - 1) * 1440) + ($$4 * 60) + $$5}')

.PHONY: help
help:
	@echo "Targets:"
	@echo "  make deps        Download Go module dependencies"
	@echo "  make test        Run the API client tests"
	@echo "  make web-build   Build the web app Docker image (frontend + FrankenPHP + Go extension)"
	@echo "  make web-up      Build and start the web app (docker compose up -d)"
	@echo "  make web-down    Stop the web app"
	@echo "  make web-logs    Follow the web app logs"
	@echo ""
	@echo "Configuration:"
	@echo "  VERSION                    Bake an app version into the image (defaults to YY.MM.ZZZZZ)"
	@echo ""
	@echo "Examples:"
	@echo "  make web-build VERSION=26.09.00001"
	@echo "  make web-up"

.PHONY: deps
deps:
	$(GO) mod download

.PHONY: test
test:
	$(GO) test ./apiclients/...

# --- Web app (FrankenPHP + Slim 4 + Vue) -------------------------------------

.PHONY: web-build
web-build:
	$(COMPOSE) build --build-arg VERSION=$(VERSION)

.PHONY: web-up
web-up: web-build
	$(COMPOSE) up -d

.PHONY: web-down
web-down:
	$(COMPOSE) down

.PHONY: web-logs
web-logs:
	$(COMPOSE) logs -f

.PHONY: web-test
web-test: test
