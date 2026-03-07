# Fly.io Deployment Guide

This guide explains how to deploy the Fucci API service to Fly.io.

## Prerequisites

1. [Install Fly.io CLI](https://fly.io/docs/getting-started/installing-flyctl/)
2. [Create a Fly.io account](https://fly.io/app/sign-up)
3. Authenticate with Fly.io:
   ```bash
   flyctl auth login
   ```

## Initial Setup

### 1. Create PostgreSQL Database

**Note:** Fly.io recommends using Managed Postgres (`fly mpg`) instead of unmanaged Postgres for production use. For managed Postgres:

```bash
cd services/api
fly mpg create --name fucci-api-db --region iad
```

**OR** for unmanaged Postgres (not supported by Fly.io Support):

```bash
cd services/api
flyctl postgres create --name fucci-api-db --region iad
```

**Important:** App names must:

- Use only lowercase letters, numbers, and dashes
- Be under 63 characters
- Example: `fucci-api-db` ✅ (valid), `fucci_db` ❌ (invalid - underscores not allowed)

Save the connection string that's displayed. You'll need it for the `DB_URL` environment variable.

### 2. Create Redis Instance

```bash
flyctl redis create --name fucci-api-redis --region iad
```

**Important:** App names must use only lowercase letters, numbers, and dashes (no underscores).

Save the connection string for the `REDIS_URL` environment variable.

### 3. Create the App

**Important:** You must create the app before setting secrets or attaching databases.

```bash
cd services/api
flyctl launch --no-deploy
```

This will:

- Create the app on Fly.io (if it doesn't exist)
- Use the existing `fly.toml` file (already included in the repo)
- **NOT** deploy the app yet (we'll do that after setting up secrets)

**Alternative:** If you want to create the app with a specific name:

```bash
flyctl apps create fucci-api
```

**Note:** The app name in `fly.toml` must match the app name you create. The default in `fly.toml` is `fucci-api`.

### 4. Set Environment Variables

**Important:** Wait until after attaching database and Redis (steps 5-6) before setting all secrets, as Fly.io sets some automatically.

Set all required environment variables:

```bash
# Generate JWT Secret (do this first)
openssl rand -base64 32

# Set all secrets
flyctl secrets set \
  DB_URL="<postgres-connection-string>" \
  REDIS_URL="<redis-connection-string>" \
  FOOTBALL_API_KEY="<your-football-api-key>" \
  RAPID_API_KEY="<your-rapidapi-key>" \
  OPENAI_API_KEY="<your-openai-api-key>" \
  JWT_SECRET="<generated-jwt-secret>" \
  --app fucci-api

# Optional
flyctl secrets set \
  OPENAI_BASE_URL="https://api.openai.com/v1" \
  PORT="8080" \
  ENVIRONMENT="production" \
  --app fucci-api
```

**Note:** When you attach PostgreSQL and Redis in the next steps, Fly.io may automatically set `DATABASE_URL` and `REDIS_PRIVATE_URL`. You can use those values or set `DB_URL` and `REDIS_URL` explicitly as shown above.

### 5. Attach PostgreSQL to the App

**For Managed Postgres (`fly mpg`):**

```bash
fly mpg attach fucci-api-db --app fucci-api
```

**For Unmanaged Postgres:**

```bash
flyctl postgres attach --app fucci-api fucci-api-db
```

This automatically sets the `DATABASE_URL` environment variable. You may need to update your code to use `DATABASE_URL` or create `DB_URL` from it:

```bash
# Get the DATABASE_URL from the attached database
flyctl secrets list --app fucci-api
# Then set DB_URL if your code uses that instead
flyctl secrets set DB_URL="<value-from-DATABASE_URL>" --app fucci-api
```

### 6. Attach Redis to the App

```bash
flyctl redis attach --app fucci-api fucci-api-redis
```

This may automatically set `REDIS_URL` or `REDIS_PRIVATE_URL`.

**Verify Redis URL is set:**

```bash
flyctl secrets list --app fucci-api
```

**If Redis URL is not set or you need to set it manually:**

```bash
# Get Redis private URL (for internal connections)
flyctl redis status fucci-api-redis

# Set REDIS_URL explicitly (use private URL for internal connections)
# Format: redis://internal.<redis-app-name>.internal:6379
flyctl secrets set REDIS_URL="redis://internal.fucci-api-redis.internal:6379" --app fucci-api
```

**Note:** Use the private/internal URL for Redis connections within Fly.io's network for better security and performance.

## Deployment

### Manual Deployment

```bash
cd services/api
flyctl deploy
```

### Automatic Deployment via GitHub Actions

The repository includes a GitHub Actions workflow (`.github/workflows/deploy-api.yml`) that automatically deploys to Fly.io when changes are pushed to the `main` or `master` branch.

**Setup GitHub Secrets:**

1. Go to your GitHub repository settings
2. Navigate to Secrets and variables > Actions
3. Add the following secret:
   - `FLY_API_TOKEN`: Your Fly.io API token (get it from `flyctl auth token`)

**Deployment will trigger automatically on:**

- Push to `main` or `master` branch
- Changes to `services/api/**` files
- Manual workflow dispatch

## Health Checks

The app includes a health check endpoint at `/v1/api/health`. Fly.io is configured to check this endpoint automatically.

## Monitoring

### View Logs

```bash
flyctl logs --app fucci-api
```

### Check App Status

```bash
flyctl status --app fucci-api
```

### SSH into the Machine

```bash
flyctl ssh console --app fucci-api
```

## Scaling

### Scale Vertically (CPU/Memory)

```bash
flyctl scale vm shared-cpu-1x --vm-memory 1024 --app fucci-api
```

### Scale Horizontally (Number of Instances)

```bash
flyctl scale count 2 --app fucci-api
```

## Database Migrations

Migrations run **automatically** on each deploy via Fly’s `release_command` in `fly.toml`. Before the new release is switched, Fly runs `./migrate` in the app image (using `DB_URL` from secrets). No manual step is required for normal deploys.

**Deploy (migrations run on Fly during release):**

```bash
# From repo root
./scripts/deploy-api.sh
# or
yarn deploy:backend
```

**Optional: run migrations locally before deploy** (e.g. against a staging DB):

```bash
RUN_MIGRATIONS=1 ./scripts/deploy-api.sh
```

**Run migrations only** (e.g. local or CI against a specific DB):

```bash
yarn migrate   # uses DB_URL from environment
```

**Manual run inside Fly container** (if needed):

```bash
flyctl ssh console --app fucci-api
# Then run: ./migrate
```

## Rollback

If you need to rollback to a previous deployment:

```bash
flyctl releases list --app fucci-api
flyctl releases rollback <release-id> --app fucci-api
```

## Environment Variables Reference

| Variable           | Required | Description                                              |
| ------------------ | -------- | -------------------------------------------------------- |
| `DB_URL`           | Yes      | PostgreSQL connection string                             |
| `REDIS_URL`        | Yes      | Redis connection string                                  |
| `FOOTBALL_API_KEY` | Yes      | API key for football data                                |
| `RAPID_API_KEY`    | Yes      | RapidAPI key                                             |
| `OPENAI_API_KEY`   | Yes      | OpenAI API key                                           |
| `JWT_SECRET`       | Yes      | Secret key for JWT tokens                                |
| `OPENAI_BASE_URL`  | No       | OpenAI API base URL (default: https://api.openai.com/v1) |
| `PORT`             | No       | Server port (default: 8080)                              |
| `ENVIRONMENT`      | No       | Environment name (default: production)                   |

## Troubleshooting

### App won't start

1. Check logs: `flyctl logs --app fucci-api`
2. Verify environment variables: `flyctl secrets list --app fucci-api`
3. Check health endpoint: `curl https://fucci-api.fly.dev/v1/api/health`

### Database connection issues

1. Verify PostgreSQL is attached: `flyctl postgres list`
2. Check connection string format
3. Ensure SSL mode is set correctly

### Redis connection issues

1. Verify Redis is attached: `flyctl redis list`
2. Check connection string format
3. Verify Redis instance is running: `flyctl redis status fucci-api-redis`

## Support

For more information, see:

- [Fly.io Documentation](https://fly.io/docs/)
- [Fly.io Go Guide](https://fly.io/docs/languages-and-frameworks/golang/)
