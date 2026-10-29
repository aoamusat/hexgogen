# Hexgogen

Hexgogen generates Go API projects that follow **hexagonal architecture (ports
and adapters)**, with a complete example feature and an embedded, versioned
application template. The CLI uses the Go standard library; application
dependencies belong to the generated project.

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
existing directory containing no application files (`--dir` defaults to the current
directory). An existing `.git` directory or worktree metadata file is allowed and
preserved. Other dotfiles count as existing content; symlink destinations are
refused. Neither command merges into an existing project or overwrites files.

The module defaults to the project directory name, so `hexgogen new myapp` works
for local development. Set `--module` to your repository's module path when
sharing or publishing the project. Names start with a letter or digit and use
letters, digits, dots, hyphens, or underscores. Paths and reserved device names
are rejected. Generation validates and renders files before creating a new
directory, confines writes to the destination, and cleans up its own partial
output on a write failure.

Generation works offline. After writing the project, the CLI runs `git init` if
the destination is not already part of a repository. Existing repositories,
including parent repositories and linked worktrees, are preserved. Generated
files remain unstaged, ready for your first commit. If Git is missing or setup
fails, the project is still generated and a warning explains the next step.

Go module downloads, containers, and database migrations are separate setup steps
explained in the generated README.


## Download a release binary

Published [GitHub Releases](https://github.com/aoamusat/hexgogen/releases) include
Linux, macOS (`darwin`), and Windows binaries for `amd64` and `arm64`. Download the
archive for your OS and CPU together with `checksums.txt`. Archives include the
CLI and README; the application template is embedded in the binary. Go is not
required to run the CLI, but is required to build and run generated applications.

For example, on Linux amd64 after downloading a release:

```sh
sha256sum --ignore-missing -c checksums.txt
tar -xzf hexgogen_v0.1.0_linux_amd64.tar.gz
sudo install -m 0755 hexgogen /usr/local/bin/hexgogen
hexgogen version
```

Replace `v0.1.0` with the release you downloaded. On macOS, extract the matching
`.tar.gz` and install the binary into a directory on PATH. On Windows, extract the
`.zip` and put `hexgogen.exe` in a directory on PATH.

## Publishing releases

Push a semantic version tag after committing the release changes:

```sh
git tag -a v0.1.0 -m "Hexgogen v0.1.0"
git push origin v0.1.0
```

The [release workflow](.github/workflows/release.yml) runs the reusable CI checks,
including PostgreSQL integration tests, then cross-compiles all six platform
binaries. It uploads `.tar.gz` archives for Linux/macOS and `.zip` archives for
Windows as workflow artifacts. After every build succeeds, it verifies SHA-256
checksums, creates a draft GitHub Release, uploads the archives and combined
`checksums.txt`, and publishes the release. Tags such as `v0.1.0-rc.1` produce
prereleases. The workflow uses `GITHUB_TOKEN`; no separate publishing secret is
needed.

A failed upload leaves a draft that can be retried by rerunning the workflow.
Already-published release assets are preserved; use a new tag for changes. The
release binaries report their tag in `hexgogen version`; `go install ...@v0.1.0`
uses the module's embedded build information to report its version as well.

For local packaging on Linux with Go, Bash, tar, zip, and sha256sum installed:

```sh
bash scripts/build-release.sh v0.1.0 linux amd64 dist
```

## Generated application

The generated codebase follows hexagonal architecture: domain interfaces define
the ports, Gin handlers provide the HTTP adapter, and PostgreSQL repositories
provide the persistence adapter. Services implement application use cases against
the repository interfaces, keeping them independent of the database implementation.
The user example demonstrates the flow from handler to service to repository.

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
