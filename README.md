# goshort — URL shortening service in Go

A self-hosted URL shortener with visit tracking and referrer detection. Built with Go, SQLite (via GORM), and Docker.

---

## Table of Contents

- [Features](#features)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Build from Source](#build-from-source)
- [Run the Service](#run-the-service)
- [API Reference](#api-reference)
- [Authentication](#authentication)
- [Migrate from BoltDB to SQLite](#migrate-from-boltdb-to-sqlite)
- [SQLite Database Queries](#sqlite-database-queries)

---

## Features

- Shorten URLs with auto-generated hashids (minimum 5 characters)
- HTTP 301 redirect on short URL access
- Track visits per short URL with referrer parsing
- Detect traffic source: ads, email, social, search, direct, internal, or link
- Aggregated visit statistics by referrer type and date
- HTTP Basic Auth or API Key Auth for protected endpoints
- One-time migration utility from BoltDB to SQLite

---

## Quick Start

```bash
cd docker
cp dot-env.example .env
# Edit .env with your credentials
docker compose up
```

The service will be available at `http://localhost:33512`.

The SQLite database file will be created at `./docker/goshort.sqlite` (based on `DATABASE_NAME` in `.env`).

---

## Configuration

All configuration is done via environment variables, loaded from `./docker/.env`.

| Variable           | Required | Default  | Description |
|--------------------|----------|----------|-------------|
| `LISTEN_PORT`      | Yes      | `33512`  | HTTP server port |
| `DATABASE_TYPE`    | Yes      | `sqlite` | Database type (only `sqlite` is currently supported) |
| `DATABASE_NAME`    | Yes      | `goshort`| SQLite database filename (stored as `{name}.sqlite`) |
| `AUTH_TYPE`        | No       | (basic)  | Set to `apikey` to use API key auth; omit for HTTP Basic Auth |
| `AUTH_USERNAME`    | Yes*     | —        | Username for HTTP Basic Auth (*required when `AUTH_TYPE` is not `apikey`) |
| `AUTH_PASSWORD`    | Yes*     | —        | Password for HTTP Basic Auth (*required when `AUTH_TYPE` is not `apikey`) |
| `AUTH_APIKEY`      | Yes*     | —        | API key value (*required when `AUTH_TYPE=apikey`) |
| `DATABASE_MIGRATE` | No       | `false`  | Set to `true` to run BoltDB → SQLite migration on startup |
| `TIMEZONE`         | No       | `UTC`    | IANA timezone name for visit date grouping (e.g. `Asia/Taipei`) |

Example `.env`:

```dotenv
AUTH_USERNAME=admin
AUTH_PASSWORD=changeme
LISTEN_PORT=33512
DATABASE_TYPE=sqlite
DATABASE_NAME=goshort
DATABASE_MIGRATE=false
TIMEZONE=Asia/Taipei
```

---

## Build from Source

This builds a Docker image from the local source code and tags it as `netivism/goshort:local`.

```bash
cd docker
docker compose -f docker-compose-src.yml build
```

The database file will be located at `./docker/<DATABASE_NAME>.sqlite`.

---

## Run the Service

### Run from locally built image

After building from source (see above):

```bash
cd docker
docker compose -f docker-compose-src.yml up
```

### Run from remote image

Uses the pre-built image `netivism/goshort:sqlite` from Docker Hub:

```bash
cd docker
docker compose up
```

See `./docker/docker-compose.yml` for details.

---

## API Reference

### Authentication

Protected endpoints require either HTTP Basic Auth or API Key Auth depending on your `AUTH_TYPE` setting.

**HTTP Basic Auth (default):**
```
Authorization: Basic <base64(username:password)>
```

```bash
curl -u username:password https://your-domain/handle/create \
  -H "Content-Type: application/json" \
  -d '{"redirect": "https://example.com/some/long/path"}'
```

**API Key Auth (`AUTH_TYPE=apikey`):**
```
Authorization: Bearer <your-api-key>
```

```bash
curl -H "Authorization: Bearer <your-api-key>" \
  -H "Content-Type: application/json" \
  https://your-domain/handle/create \
  -d '{"redirect": "https://example.com/some/long/path"}'
```

---

### POST /handle/create

Create a new shortened URL.

**Auth required:** Yes

**Request body:**
```json
{
  "redirect": "https://example.com/some/long/path"
}
```

**Response `201 Created`:**
```json
{
  "success": 1,
  "message": "URL shorten successfully.",
  "result": [
    {
      "short": "aB3xY",
      "redirect": "https://example.com/some/long/path",
      "count": 1
    }
  ]
}
```

**Validation:**
- URL must be a valid `http://` or `https://` URL
- URL must not contain embedded username or password

---

### GET /{id}

Redirect a short URL to its original target. This is the public endpoint — no authentication required.

**Example:**
```
GET /aB3xY
→ HTTP 301 → https://example.com/some/long/path
```

Each access records a visit entry with referrer information.

---

### GET /handle/visits/{id}

Get aggregated visit statistics for a specific short URL.

**Auth required:** Yes

**Response `200 OK`:**
```json
{
  "success": 1,
  "message": "Visits loaded successfully.",
  "result": {
    "total": 7,
    "referrer_statistics": {
      "social": {
        "all": 5,
        "facebook": 3,
        "twitter": 2
      },
      "search": {
        "all": 2,
        "google": 2
      }
    },
    "dates": {
      "2024-01-15": {
        "allday": 120,
        "0": 3,
        "1": 0,
        "2": 1,
        "...": "...",
        "23": 5
      },
      "2024-01-16": {
        "allday": 8
      }
    }
  }
}
```

**Notes:**
- `total` is the total number of visits recorded for this short URL.
- `referrer_statistics` groups visits by type (e.g. `social`, `search`, `ad`, `email`, `direct`, `link`, `internal`, `unknown`). Each type contains an `all` count plus per-network breakdowns (e.g. `facebook`, `google`).
- `dates` groups visits by calendar date (`YYYY-MM-DD`) in the timezone set by `TIMEZONE` (defaults to UTC). Each date always contains `allday` (total). Hourly keys (`0`–`23`) are included only when `allday > 10`.

**Referrer types:** `ad`, `email`, `social`, `search`, `internal`, `direct`, `link`, `unknown`

---

## Migrate from BoltDB to SQLite

If you are upgrading from an older version of goshort that used BoltDB (`goshort.db`), follow these steps to migrate your data to SQLite.

### Prerequisites

- Your existing `goshort.db` (BoltDB file) must be placed in the `./docker/` directory alongside `.env`
- SQLite database will be created at `./docker/goshort.sqlite`

### Migration Steps

**1. Place your BoltDB file in the docker directory:**
```bash
cp /path/to/your/goshort.db ./docker/goshort.db
```

**2. Enable migration mode in `.env`:**
```dotenv
DATABASE_MIGRATE=true
DATABASE_TYPE=sqlite
DATABASE_NAME=goshort
```

**3. Run the service — it will migrate and exit:**
```bash
cd docker
docker compose up
```

The container will:
1. Read all records from `goshort.db` (BoltDB)
2. Insert them into `goshort.sqlite` in batches of 1000
3. Exit automatically when migration is complete

**4. Disable migration mode after completing:**
```dotenv
DATABASE_MIGRATE=false
```

**5. Run the service normally:**
```bash
docker compose up
```

### Verifying the Migration

After migration, you can verify the record count:

```bash
# Count redirects migrated
sqlite3 ./docker/goshort.sqlite "SELECT COUNT(*) FROM redirects;"

# Preview migrated records
sqlite3 ./docker/goshort.sqlite "SELECT id, redirect, created_at FROM redirects LIMIT 10;"
```

---

## SQLite Database Queries

The SQLite database file is located at `./docker/<DATABASE_NAME>.sqlite` (e.g., `./docker/goshort.sqlite`).

### Open the database

```bash
# Using the sqlite3 CLI
sqlite3 ./docker/goshort.sqlite

# Or with Docker (if sqlite3 is not installed locally)
docker run --rm -it -v "$(pwd)/docker:/data" keinos/sqlite3 sqlite3 /data/goshort.sqlite
```

### Show all tables

```sql
.tables
-- redirects  statistics  visits
```

### Inspect table schema

```sql
.schema redirects
.schema visits
.schema statistics
```

### Query short URL records

```sql
-- List all shortened URLs
SELECT id, redirect, domain, path, created_at FROM redirects;

-- Search by domain
SELECT id, redirect FROM redirects WHERE domain = 'example.com';

-- Most recently created
SELECT id, redirect, datetime(created_at, 'unixepoch', 'localtime') AS created
FROM redirects
ORDER BY created_at DESC
LIMIT 20;
```

### Query visit records

```sql
-- All visits for a specific short URL
SELECT * FROM visits WHERE redirect_id = 'aB3xY';

-- Visit count per short URL
SELECT redirect_id, COUNT(*) AS visit_count
FROM visits
GROUP BY redirect_id
ORDER BY visit_count DESC;

-- Visits by traffic source type
SELECT referer_type, COUNT(*) AS count
FROM visits
GROUP BY referer_type
ORDER BY count DESC;

-- Visits by network (e.g., google, facebook)
SELECT referer_network, COUNT(*) AS count
FROM visits
WHERE referer_network != ''
GROUP BY referer_network
ORDER BY count DESC;

-- Visits within a time range (Unix timestamps)
SELECT * FROM visits
WHERE created_at BETWEEN 1711900800 AND 1714492800;

-- Recent visits with human-readable timestamp
SELECT redirect_id, referer_type, referer_network,
       datetime(created_at, 'unixepoch', 'localtime') AS visited_at
FROM visits
ORDER BY created_at DESC
LIMIT 50;
```

### Exit sqlite3

```sql
.quit
```
