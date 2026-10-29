# Hexgogen

A Go project starter using hexagonal architecture, Gin, PostgreSQL/GORM,
Zerolog, and PASETO v2 local tokens.

## Getting started

Install Go 1.25.6 or newer, PostgreSQL, and Make. Create a local database and
configure the application:

```sh
createdb -h localhost -U postgres hexgogen
cp .env.example .env
# Edit .env to match your local database credentials.
go mod download
make run
```

The server defaults to port 8000 and requires a reachable PostgreSQL database.
`GET /v1/users` is an empty example handler; user operations and authentication
are placeholders for your application. They are not implemented or protected yet.

For a new project, change the module path in `go.mod` and replace
`github.com/aoamusat/hexgogen` in Go imports with your own module path.

## Structure

```text
cmd/                  Entry point and process signal handling
configs/              Environment loading and validation
api/                  HTTP server, routes, middleware, and v1 handlers
domain/              Entities, DTOs, service and repository contracts
internal/adapter/     Service implementations and PostgreSQL adapters
pkg/                  Reusable logging, token, and response helpers
migrations/           Versioned SQL migrations
```

Requests flow from a Gin handler through a domain service interface to a service
implementation, then through a repository interface to the database adapter.
Most user contracts and models are intentionally empty; define their methods and
fields when adding your first feature.

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `ENV` | `development` | `development` or `production` |
| `PORT` | `8000` | HTTP port, from 1 to 65535 |
| `DATABASE_URL` | Required | PostgreSQL connection URL |

In development, `.env` is loaded before configuration is read. Existing process
environment values take precedence. A missing `.env` is fine; malformed files
and invalid configuration fail startup. Set `ENV=production` in the deployment
environment to skip `.env` loading.

Application and request logs use timestamped Zerolog JSON on stdout. Panics
produce an error log and HTTP 500 response. CORS allows all origins in both
environments without credentialed requests; restrict origins and explicitly
configure credentials if your application needs cookie-based cross-origin auth.

SIGINT and SIGTERM trigger graceful HTTP shutdown with a five-second deadline,
followed by closing the database pool. Listener failures exit with a nonzero
status. Build with `make build` and run `./bin/hexgogen` for deployments so signals
reach the application process directly.

## Migrations

Install the pinned [golang-migrate CLI](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate):

```sh
make tools
make migrate-create NAME=create_users
# Add your schema SQL to the generated up and down files.
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/hexgogen?sslmode=disable'
make migrate-up
# To roll back the most recent migration:
make migrate-down
```

Migration commands read the exported `DATABASE_URL`; they do not load `.env`.
There is no initial application schema because the domain entities are empty.
Run migrations after defining your schema and before starting code that depends
on it. The server does not run automatic schema changes. See
[migrations/README.md](migrations/README.md) for conventions.

## Checks

```sh
make check
```

This checks formatting, runs `go vet` and tests with the race detector, and builds
the application. Unit tests do not require PostgreSQL. GitHub Actions runs these
checks on pushes and pull requests using the Go version in `go.mod`.
