#### Postgres Config
This formula has created a default database cluster with:
  initdb --locale=C -E UTF-8 /usr/local/var/postgresql@16
For more details, read:
  https://www.postgresql.org/docs/16/app-initdb.html

postgresql@16 is keg-only, which means it was not symlinked into /usr/local,
because this is an alternate version of another formula.

If you need to have postgresql@16 first in your PATH, run:
  echo 'export PATH="/usr/local/opt/postgresql@16/bin:$PATH"' >> ~/.profile

For compilers to find postgresql@16 you may need to set:
  export LDFLAGS="-L/usr/local/opt/postgresql@16/lib"
  export CPPFLAGS="-I/usr/local/opt/postgresql@16/include"

To start postgresql@16 now and restart at login:
  brew services start postgresql@16
Or, if you don't want/need a background service you can just run:
  LC_ALL="C" /usr/local/opt/postgresql@16/bin/postgres -D /usr/local/var/postgresql@16
  
#### DB Commands (migrations)
    From repo root (uses DB_URL from .env):
      yarn migrate
    Or from services/api with DB_URL set:
      go run -tags migrate ./cmd/migrate
    Migrations live in services/api/sql/schema/ and use -- +goose Up / -- +goose Down markers.


##### Creating New Model
    - Step 1) Create New Schema
    - Step 2) Create Query
    - Step 3) Run `sqlc generate`

#### Google OAuth Environment

Set these values in `.env` (or exported env vars) before enabling Google auth endpoints:

- `GOOGLE_OAUTH_CLIENT_ID` - Google OAuth web client ID used for ID token audience validation.
- `GOOGLE_OAUTH_CLIENT_SECRET` - OAuth client secret used for auth code exchange.
- `GOOGLE_OAUTH_REDIRECT_URIS` - Comma-separated allowlist for callback URIs (e.g. `fucci://auth,com.fucci.app:/oauth2redirect`).
