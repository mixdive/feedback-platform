.PHONY: build test swag-init docker tidy dev-mongo dev-mongo-down dev-mongo-reset web-build web-install image

SWAG := $(shell command -v swag 2>/dev/null || echo $(shell go env GOPATH)/bin/swag)
BUN  := $(shell command -v bun 2>/dev/null  || echo $(HOME)/.bun/bin/bun)

# Build the binary. Always builds the React apps first and embeds them; we
# only ever ship one mode (frontend + backend together).
#
# Symbol table + DWARF info are intentionally kept (no `-ldflags "-s -w"`)
# so VS Code's F5 launch under dlv can walk goroutines. Production builds
# happen through `make image`, where the Dockerfile applies the strip
# flags itself, so the shipped binary stays small.
build: web-build
	go build -trimpath -o bin/mixdive .

test:
	go test ./...

tidy:
	go mod tidy

# Regenerate swag annotations into docs/api/. Generated files are committed.
swag-init:
	@if [ ! -x "$(SWAG)" ]; then \
		echo "swag CLI not found at $(SWAG); installing..."; \
		go install github.com/swaggo/swag/cmd/swag@latest; \
	fi
	"$(SWAG)" init --parseDependency --parseInternal \
		--dir ./,./api,./api/console,./api/portal,./api/middlewares,./api/response,./dataoperations,./models \
		-g main.go -o docs/api

# Install React dependencies for both apps.
web-install:
	"$(BUN)" install --cwd web/console
	"$(BUN)" install --cwd web/portal

# Build both React apps to web/{console,portal}/dist/.
web-build: web-install
	cd web/console && "$(BUN)" run build
	cd web/portal  && "$(BUN)" run build

# Build the production Docker image (multi-stage).
image:
	docker build -t mixdive/feedback-platform:0.1.0 .

# Stub kept for backwards compatibility — alias of `image`.
docker: image

# Local Mongo replica set for development. See docker-compose.dev.yml.
dev-mongo:
	docker compose -f docker-compose.dev.yml up -d
	@echo "Mongo: mongodb://localhost:27017/?replicaSet=rs0"

dev-mongo-down:
	docker compose -f docker-compose.dev.yml down

dev-mongo-reset:
	docker compose -f docker-compose.dev.yml down -v
