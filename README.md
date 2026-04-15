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
  - [POST /handle/create](#post-handlecreate)
  - [POST /handle/batch-create](#post-handlebatch-create)
  - [POST /handle/batch-info](#post-handlebatch-info)
  - [GET /{id}](#get-id)
  - [GET /handle/visits/{id}](#get-handlevisitsid)
- [Authentication](#authentication)
- [Statistics Cache & Monthly Cleanup](#statistics-cache--monthly-cleanup)
- [Migrate from BoltDB to SQLite](#migrate-from-boltdb-to-sqlite)
- [SQLite Database Queries](#sqlite-database-queries)

---

## Features

- Shorten URLs with auto-generated hashids (minimum 5 characters)
- HTTP 301 redirect on short URL access
- Track visits per short URL with referrer parsing
- Detect traffic source: ads, email, social, search, direct, internal, or link
- Aggregated visit statistics by referrer type and date, with per-hour breakdown
- Statistics caching: results are cached in the `statistics` table and refreshed at most once per hour
- Incremental aggregation: only new visits (since last cache update) are processed on each refresh, preserving historical data even after the `visits` table is cleared
- Automatic monthly cleanup: visits older than the current month are deleted on the 1st of each month at 02:00 (configured timezone), after statistics are refreshed and persisted
- HTTP Basic Auth or API Key Auth for protected endpoints
- One-time migration utility from BoltDB to SQLite (visit counts are preserved in `statistics`)

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

| Variable                   | Required | Default  | Description |
|----------------------------|----------|----------|-------------|
| `LISTEN_PORT`              | Yes      | `33512`  | HTTP server port |
| `DATABASE_TYPE`            | Yes      | `sqlite` | Database type (only `sqlite` is currently supported) |
| `DATABASE_NAME`            | Yes      | `goshort`| SQLite database filename (stored as `{name}.sqlite`) |
| `AUTH_TYPE`                | No       | (basic)  | Set to `apikey` to use API key auth; omit for HTTP Basic Auth |
| `AUTH_USERNAME`            | Yes*     | —        | Username for HTTP Basic Auth (*required when `AUTH_TYPE` is not `apikey`) |
| `AUTH_PASSWORD`            | Yes*     | —        | Password for HTTP Basic Auth (*required when `AUTH_TYPE` is not `apikey`) |
| `AUTH_APIKEY`              | Yes*     | —        | API key value (*required when `AUTH_TYPE=apikey`) |
| `DATABASE_MIGRATE`         | No       | `false`  | Set to `true` to run BoltDB → SQLite migration on startup |
| `TIMEZONE`                 | No       | `UTC`    | IANA timezone name for visit date grouping and monthly cleanup schedule (e.g. `Asia/Taipei`) |
| `VISITS_HOURLY_THRESHOLD`  | No       | `10`     | Controls hourly breakdown in API output: `0` = never show, `1` = always show, `N > 1` = show only when `allday > N`. Hourly data is always stored internally regardless of this setting. |

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

### POST /handle/batch-create

Create multiple shortened URLs in a single request. Up to 1000 entries per call.

**Auth required:** Yes

**Request body:** JSON array of objects, each with a `redirect` field.
```json
[
  {"redirect": "https://example.com/page1"},
  {"redirect": "https://example.com/page2"},
  {"redirect": "https://example.com/page3"}
]
```

**Response `201 Created`:**
```json
{
  "success": 1,
  "message": "3/3 URLs shortened successfully.",
  "result": [
    {"redirect": "https://example.com/page1", "short": "aB3xY"},
    {"redirect": "https://example.com/page2", "short": "cD4zA"},
    {"redirect": "https://example.com/page3", "short": "eF5bB"}
  ]
}
```

If some entries fail to save (e.g. due to a database error), they are included in the result **without** a `short` field and with an `error` field instead. The remaining entries are still processed.

```json
{
  "success": 1,
  "message": "2/3 URLs shortened successfully. 1 failed.",
  "result": [
    {"redirect": "https://example.com/page1", "short": "aB3xY"},
    {"redirect": "https://example.com/page2", "short": "cD4zA"},
    {"redirect": "https://example.com/page3", "error": "error saving record: ..."}
  ]
}
```

**Validation (applied to all entries before any insert):**
- Each entry must have a `redirect` field
- URL must be a valid `http://` or `https://` URL
- URL must not contain embedded username or password
- Maximum 1000 entries per request

---

### POST /handle/batch-info

Look up redirect targets and total visit counts for multiple short URL IDs in a single request.

**Auth required:** Yes

**Request body:** JSON array of short URL IDs.
```json
["aB3xY", "cD4zA", "eF5bB"]
```

**Response `200 OK`:**
```json
{
  "success": 1,
  "message": "Redirect info loaded successfully.",
  "result": [
    {"id": "aB3xY", "redirect": "https://example.com/page1", "total": 42},
    {"id": "cD4zA", "redirect": "https://example.com/page2", "total": 7},
    {"id": "eF5bB", "redirect": "",                          "total": 0}
  ]
}
```

**Notes:**
- The response preserves the same order as the input array.
- If an ID does not exist in the database, `redirect` is an empty string and `total` is `0`. The entry is still included in the result.
- `total` is read from the `statistics` cache. It reflects the cumulative count as of the last statistics refresh. It does **not** trigger a fresh aggregation — call `GET /handle/visits/{id}` to recompute.

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

**Query parameters:**

| Parameter | Value | Description |
|-----------|-------|-------------|
| `refresh` | `1`   | Force bypass the 1-hour statistics cache and recompute immediately. Omit for normal cached behaviour. |

**Examples:**
```bash
# Normal request (returns cached result if available)
curl -H "Authorization: Bearer <key>" https://your-domain/handle/visits/aB3xY

# Force refresh (bypasses cache, recomputes from visits table)
curl -H "Authorization: Bearer <key>" https://your-domain/handle/visits/aB3xY?refresh=1
```

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
- `total` is the cumulative visit count, including historical data preserved in the `statistics` table even after the `visits` table has been cleared.
- `referrer_statistics` groups visits by type (e.g. `social`, `search`, `ad`, `email`, `direct`, `link`, `internal`, `unknown`). Each type contains an `all` count plus per-network breakdowns (e.g. `facebook`, `google`).
- `dates` groups visits by calendar date (`YYYY-MM-DD`) in the timezone set by `TIMEZONE` (defaults to UTC). Each date always contains `allday` (total). Hourly keys (`0`–`23`) are included in the response based on `VISITS_HOURLY_THRESHOLD`: `0` = never, `1` = always, `N > 1` = only when `allday > N` (defaults to `10`). Hourly data is **always stored** in the statistics cache regardless of this threshold.
- Results are served from the `statistics` cache when it was updated within the last hour. A cache miss triggers incremental aggregation (only visits newer than the last cache update are processed) and updates the cache. Use `?refresh=1` to force a recompute regardless of cache age.

**Referrer types:** `ad`, `email`, `social`, `search`, `internal`, `direct`, `link`, `unknown`

---

## Statistics Cache & Monthly Cleanup

### How it works

Visit statistics are stored in two tables:

| Table        | Purpose | Retention |
|--------------|---------|-----------|
| `visits`     | Raw per-connection records | Cleared monthly by the automatic cleanup job |
| `statistics` | Aggregated result cache (one row per short URL) | Kept indefinitely; accumulates history across cleanups |

**Aggregation flow:**

1. `GET /handle/visits/{id}` is called.
2. If `statistics.update_at` is within the last hour → return the cached result immediately.
3. Otherwise, query only `visits` records newer than `statistics.agg_date_end` (incremental), merge them with the existing cached result, and update the cache.

Because the incremental query uses `agg_date_end` as the watermark, historical data in `statistics` is never lost when `visits` is cleared.

**Storage vs. output:**

- The `statistics` table always stores full per-hour data (`0`–`23`) for every date.
- The API response applies `VISITS_HOURLY_THRESHOLD` at output time, so clients only receive hourly detail when the day's traffic exceeds the configured threshold.

### Monthly automatic cleanup

At **02:00 on the 1st of each month** (in the timezone set by `TIMEZONE`), the service automatically:

1. Identifies all short URL IDs that have visit records older than the current month.
2. Calls the statistics refresh for each ID, ensuring `statistics` is fully up-to-date.
3. If all refreshes succeed, deletes all `visits` rows with `created_at` before the current month start.
4. If any refresh fails, the deletion is aborted to prevent data loss.

This keeps the `visits` table lean (≤ 1 month of raw data at any time) while `statistics` continues to accumulate totals indefinitely.

> **Note:** No manual intervention is required. The cleanup goroutine starts automatically with the service.

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
2. Insert redirect records into `goshort.sqlite` in batches of 1000
3. For each record that has a non-zero visit count, create a corresponding `statistics` row preserving the historical total (`agg_date_start` = 2020-01-01, `agg_date_end` = migration time)
4. Exit automatically when migration is complete

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

### Query statistics records

```sql
-- View cached statistics for all short URLs
SELECT redirect_id,
       json_extract(result, '$.total') AS total,
       date(agg_date_start, 'unixepoch') AS since,
       datetime(agg_date_end, 'unixepoch', 'localtime') AS last_agg,
       datetime(update_at, 'unixepoch', 'localtime') AS cache_updated
FROM statistics
ORDER BY total DESC;

-- Check if the cache is fresh (updated within the last hour)
SELECT redirect_id,
       json_extract(result, '$.total') AS total,
       CASE WHEN (strftime('%s', 'now') - update_at) < 3600
            THEN 'fresh' ELSE 'stale' END AS cache_status
FROM statistics;

-- Manually inspect the full JSON result for a specific short URL
SELECT result FROM statistics WHERE redirect_id = 'aB3xY';
```

### Exit sqlite3

```sql
.quit
```
