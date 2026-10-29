# Hexgogen

Hexgogen generates Go API projects with a complete example feature and an
embedded, versioned application template. The CLI uses the Go standard library;
application dependencies belong to the generated project.

## Build and use

Install Go 1.25.6 or newer:

```sh
go build -o bin/hexgogen ./cmd/hexgogen
./bin/hexgogen new myapp --module github.com/yourname/myapp
```

For a checked-out repository, `go install ./cmd/hexgogen` installs the CLI into
`GOBIN` (or the default Go bin directory). Add that directory to your PATH to use
`hexgogen` directly. A published release can be installed with
`go install github.com/aoamusat/hexgogen/cmd/hexgogen@<tag>`.

```sh
hexgogen new myapp
hexgogen new myapp --module github.com/yourname/myapp --dir /path/to/projects

mkdir existing-project
cd existing-project
hexgogen init --module github.com/yourname/existing-project

hexgogen --help
hexgogen version
```

`new` creates a new directory under the current directory, or under `--dir`.
It refuses an existing target, even an empty directory. `init` requires an
existing, empty directory (`--dir` defaults to the current directory). Dotfiles
count as existing content; symlink destinations are refused. Neither command
merges into an existing project or overwrites files.

The module defaults to the project directory name, so `hexgogen new myapp` works
for local development. Set `--module` to your repository's module path when
sharing or publishing the project. Names start with a letter or digit and use
letters, digits, dots, hyphens, or underscores. Paths and reserved device names
are rejected. Generation validates and renders files before creating a new
directory, confines writes to the destination, and cleans up its own partial
output on a write failure.

Generation works offline. It writes files without running Git, downloading Go
modules, starting containers, or applying database migrations. The generated
README explains the next steps.

## Generated application

Each project includes:

- A create/get-user example across handlers, services, repository interfaces,
  PostgreSQL adapters, and a versioned SQL migration.
- Consistent JSON responses, validation, application-error mapping, and request IDs.
- `/healthz` and `/readyz` endpoints.
- Zerolog JSON logs with configurable levels and structured request fields.
- Request context propagation, HTTP/body limits, database pool controls, and
  graceful SIGINT/SIGTERM shutdown.
- `.env.example`, a Makefile, local PostgreSQL Compose setup, and migration commands.
- Unit, token, and PostgreSQL integration tests, plus GitHub Actions CI.

The user endpoints are public examples; authentication and authorization remain
application decisions. A PASETO helper is included, with invalid-token tests.
See the [application README](templates/README.md) for API and configuration details.

## Repository layout

```text
cmd/hexgogen/          CLI entry point
internal/cli/         Commands, argument parsing, and output
internal/generator/   Validation, rendering, filesystem writes, and generation tests
templates/embed.go    Embedded template definition
templates/           Canonical application files alongside the embedding definition
```

Application Go source uses `.go.tmpl` suffixes, and the application module file
is stored as `go.mod.tmpl`. This keeps application packages separate from the CLI
while storing their source directly in `templates/`. Generation removes the
`.tmpl` suffixes and formats the resulting Go files. The embedding patterns include
dotfiles and CI workflows. Module paths and the README project name are replaced
during generation.

To change the application, edit `templates/`, then run the checks. There is
one canonical template, with no duplicated application source to synchronize.
To inspect a runnable copy:

```sh
go run ./cmd/hexgogen new preview --module example.com/preview --dir /tmp
cd /tmp/preview
# Follow its README to start PostgreSQL, apply migrations, and run the server.
```

## Validation

```sh
make check
```

Checks include formatting (including template Go files), vet, race tests, and a CLI
build. Generation tests verify module replacement, dotfiles, invalid inputs,
overwrite refusal, filesystem confinement, and cleanup. They also generate an
application and run its vet, race tests, and build; a local module name is built
as well. The first full run needs access to download the application's Go modules
unless they are already cached. `go test -short ./...` skips the generated
application vet/test/build checks.

For PostgreSQL integration tests, supply a dedicated test database with
schema-creation privileges:

```sh
export TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/hexgogen_test?sslmode=disable'
make integration
```

These tests generate a project, exercise its repository and HTTP user flow, and
check migration up/down/reapply in isolated, randomly named schemas. The database
must already exist. Full `make check` also runs them when `TEST_DATABASE_URL` is
set. CI provisions PostgreSQL and enables these checks. Generated projects have
their own matching CI workflow.

Build a versioned CLI with `make build VERSION=v0.1.0`. Release builds embed the
application template at build time, so a CLI version always ships its own template.
