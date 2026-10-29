# Database migrations

The sample schema starts with `000001_create_users`. Apply it before using the
user endpoints. Migrations use the [golang-migrate CLI](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate).

```sh
make tools
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/hexgogen?sslmode=disable'
make migrate-up
make migrate-create NAME=add_email
# Fill in both generated .up.sql and .down.sql files.
make migrate-up
make migrate-down # Rolls back one migration.
```

Commands use exported `DATABASE_URL` and do not load `.env`. Server startup does
not change the schema. Use transactions when all statements support them; test up
and down migrations against a disposable database. Never edit deployed migrations.

The sample migration is tested through up/down/reapply against PostgreSQL in CI.
