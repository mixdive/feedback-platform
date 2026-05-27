# Mixdive

Self-hostable, open-source feedback portal. Single Docker image, one MongoDB connection. Ships a public **Portal** for end users to submit and vote on feedback, plus an admin **Console** for triage, releases, and changelog.

---

## Features

- **Portal** — end users submit feedback, upvote entries, and comment.
- **Console** — admins triage entries (status, topic, release, internal flag), manage taxonomy, post replies, and merge duplicates.
- **Releases & changelog** — group entries into versioned releases and publish a public changelog.
- **Optional AI assistance** — when configured with an API key, Mixdive can auto-classify entry type, assign topics, and surface related entries.
- **Single binary** — Go backend with both React SPAs embedded. One Docker image, one Mongo connection string.

---

## Tech stack

- **Backend:** Go, Gin, swaggo, mongo-go-driver v2
- **Frontend:** React 19, TypeScript, Vite 7, Tailwind v4, Redux Toolkit, React Query v5
- **Database:** MongoDB (replica set required for transactions — single-node is fine)
- **Build:** Bun, multi-stage Docker

---

## Quick start

### Docker Compose (self-host)

```bash
cp .env.example .env   # fill in MONGO_URI if not using the bundled mongo service
docker compose up
```

The compose recipe brings up Mixdive and a single-node MongoDB replica set, then exposes the server on `:8080`. Open `http://localhost:8080/` for the first-run setup wizard.

### Local development

Three terminals: backend on `:8080`, Console dev server on `:5173`, Portal dev server on `:5174`.

```bash
# 1. Mongo (single-node replica set)
make dev-mongo

# 2. Build and run the Go backend
cp .env.example .env
make build
set -a; source .env; set +a
./bin/mixdive

# 3a. Console dev server (admin UI)
cd web/console && bun install && bun run dev

# 3b. Portal dev server (end-user UI)
cd web/portal && bun install && bun run dev
```

Endpoints:

- Portal — http://localhost:5174/
- Console — http://localhost:5173/console/
- API docs (Swagger) — http://localhost:8080/swagger/index.html

### Production build (single binary)

```bash
make web-build       # builds both React apps to web/{console,portal}/dist
make build           # builds Go binary with embedded assets
./bin/mixdive        # serves Portal at /, Console at /console/, API at /api
```

### Docker image

```bash
make image           # multi-stage build: bun + go + slim runtime
```

---

## Configuration

Deployment-time configuration uses a single environment variable:

| Variable | Purpose |
|---|---|
| `MONGO_URI` | MongoDB connection string (replica set required) |

Everything else — organization name, branding, the bootstrap admin account, optional AI / integration keys — is configured through the first-run setup wizard and the Console settings page.

---

## Repository layout

```
.
├── main.go            # entrypoint
├── router.go          # Gin route registration
├── api/               # HTTP handlers (one file per endpoint)
│   ├── console/       # admin endpoints
│   ├── portal/        # end-user endpoints
│   ├── middlewares/   # auth + access middlewares
│   └── response/      # response envelope helpers
├── dataoperations/    # single DataOperations struct; methods grouped per model
├── models/            # Go structs (source of truth for entity shape)
├── pkg/
│   ├── mongodb/       # generic mongo-driver helpers
│   ├── storage/       # filesystem / GCS blob storage
│   ├── aianalyzer/    # optional AI analyzers (entry type, topic, relations)
│   └── github/        # GitHub issue export client
├── web/
│   ├── console/       # React app — admins
│   └── portal/        # React app — end users
├── docs/api/          # swag-generated swagger.json/yaml/docs.go
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

---

## License

[MIT](LICENSE)

---

## Links

- Repository: https://github.com/mixdive/feedback-platform
- Issues: https://github.com/mixdive/feedback-platform/issues
