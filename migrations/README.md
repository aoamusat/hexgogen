# Database migrations

Versioned SQL migrations use the golang-migrate CLI. Install the pinned CLI with
`make tools`, then create a migration with `make migrate-create NAME=create_users`.
Fill in both generated `.up.sql` and `.down.sql` files before applying them.

Export `DATABASE_URL` in your shell, then use `make migrate-up` to apply pending
migrations or `make migrate-down` to roll back one migration. These commands do
not load `.env` automatically. See the root README for an example.

There are no application migrations yet because the domain entities are empty.
Create the first migration when defining your application's schema. Migrations
run explicitly, separately from server startup; GORM does not auto-migrate.

Use PostgreSQL transactions (`BEGIN;` / `COMMIT;`) in migration files when all
statements support them. Test both directions against a disposable database before
shipping a migration. Never edit a migration that has already been deployed.
