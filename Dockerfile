# Multi-stage Dockerfile producing a single Mixdive image.
#
# Stage 1: build both React apps with Bun.
# Stage 2: build the Go binary with the React dist/ folders embedded.
# Stage 3: distroless runtime — just the binary, no shell, no toolchain.

ARG GO_VERSION=1.25
ARG BUN_VERSION=1.3

# --- Stage 1: web -------------------------------------------------------
FROM oven/bun:${BUN_VERSION} AS web-builder
WORKDIR /repo

# Install deps first (cached when package.json + lock files don't change).
COPY web/console/package.json web/console/bun.lock* ./web/console/
COPY web/portal/package.json  web/portal/bun.lock*  ./web/portal/
RUN cd web/console && bun install --frozen-lockfile || bun install
RUN cd web/portal  && bun install --frozen-lockfile || bun install

# Copy sources and build.
COPY web/console ./web/console
COPY web/portal  ./web/portal
RUN cd web/console && bun run build
RUN cd web/portal  && bun run build

# --- Stage 2: go --------------------------------------------------------
FROM golang:${GO_VERSION} AS go-builder
WORKDIR /repo

# Module cache layer.
COPY go.mod go.sum ./
RUN go mod download

# Source.
COPY . .

# Pull the React dist/ folders from the web stage.
COPY --from=web-builder /repo/web/console/dist ./web/console/dist
COPY --from=web-builder /repo/web/portal/dist  ./web/portal/dist

# Build the binary. Always-on go:embed pulls in the SPA dist folders that
# were copied from the web stage above.
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" \
      -o /out/mixdive .

# --- Stage 3: runtime ---------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=go-builder /out/mixdive /mixdive

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/mixdive"]
