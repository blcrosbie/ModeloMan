# Postgres Migrations

ModeloMan no longer creates or mutates database schema at runtime.

## Why

- The application role should not require DDL privileges.
- Extension management (`CREATE EXTENSION`) should run in a controlled migration job.
- Startup should fail fast if schema is missing or incomplete.

## Baseline migration

Initial schema migration is provided at:

- `db/migrations/001_init.sql`
- `db/migrations/002_timescale_policies.sql`

Run it with an admin/migration role before starting ModeloMan:

```bash
psql "$DATABASE_URL_ADMIN" -f db/migrations/001_init.sql
psql "$DATABASE_URL_ADMIN" -f db/migrations/002_timescale_policies.sql
```

## Runtime behavior

On startup, ModeloMan now verifies:

- required tables exist
- `timescaledb` extension is installed

If checks fail, startup returns `FailedPrecondition` and exits.

## Recommended deployment model

1. Run migrations in CI/CD (or a one-shot migration job) with privileged credentials.
2. Run ModeloMan with a restricted app role that has no `CREATE` on schema and no extension privileges.
3. Keep migration SQL versioned in `db/migrations/`.
