#!/usr/bin/env bash
# test-batch-api.sh — manual curl tests for /handle/batch-create and /handle/batch-info
#
# Usage:
#   ./test-batch-api.sh                        # basic auth (default)
#   AUTH_TYPE=apikey ./test-batch-api.sh       # api key auth

set -euo pipefail

# ─── Config ───────────────────────────────────────────────────────────────────
HOST="${HOST:-http://localhost:33512}"
AUTH_TYPE="${AUTH_TYPE:-apikey}"
AUTH_USERNAME="${AUTH_USERNAME:-admin}"
AUTH_PASSWORD="${AUTH_PASSWORD:-changeme}"
AUTH_APIKEY="${AUTH_APIKEY:-yourkey}"
# ──────────────────────────────────────────────────────────────────────────────

# Build curl auth flags
if [ "$AUTH_TYPE" = "apikey" ]; then
  AUTH_FLAGS=(-H "Authorization: Bearer $AUTH_APIKEY")
else
  AUTH_FLAGS=(-u "$AUTH_USERNAME:$AUTH_PASSWORD")
fi

PASS=0
FAIL=0

# Colours
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# ─── Helpers ──────────────────────────────────────────────────────────────────

run_curl() {
  # run_curl <description> <expected_http_code> <method> <path> [body]
  local desc="$1"
  local expected_code="$2"
  local method="$3"
  local path="$4"
  local body="${5:-}"

  local extra_flags=()
  if [ -n "$body" ]; then
    extra_flags+=(-H "Content-Type: application/json" -d "$body")
  fi

  local response
  response=$(curl -s -w "\n__STATUS__%{http_code}" -X "$method" \
    "${AUTH_FLAGS[@]}" \
    "${extra_flags[@]}" \
    "$HOST$path")

  local http_code
  http_code=$(echo "$response" | grep '__STATUS__' | sed 's/__STATUS__//')
  local body_out
  body_out=$(echo "$response" | grep -v '__STATUS__')

  if [ "$http_code" = "$expected_code" ]; then
    echo -e "${GREEN}PASS${NC}  [$http_code] $desc"
    PASS=$((PASS + 1))
  else
    echo -e "${RED}FAIL${NC}  [got $http_code, want $expected_code] $desc"
    FAIL=$((FAIL + 1))
  fi

  # Pretty-print body if jq available, otherwise raw
  if command -v jq &>/dev/null; then
    echo "$body_out" | jq . 2>/dev/null || echo "$body_out"
  else
    echo "$body_out"
  fi
  echo
}

separator() {
  echo -e "${YELLOW}━━━ $1 ━━━${NC}"
  echo
}

# ─── Tests ────────────────────────────────────────────────────────────────────

echo
echo "Host : $HOST"
echo "Auth : $AUTH_TYPE"
echo

# ── batch-create ──────────────────────────────────────────────────────────────
separator "POST /handle/batch-create"

run_curl "3 valid URLs → 201" 201 POST /handle/batch-create \
  '[
    {"redirect":"https://www.google.com"},
    {"redirect":"https://www.github.com"},
    {"redirect":"https://www.wikipedia.org"}
  ]'

run_curl "empty array → 400" 400 POST /handle/batch-create \
  '[]'

run_curl "one entry with empty redirect → 400" 400 POST /handle/batch-create \
  '[{"redirect":""}]'

run_curl "ftp:// scheme → 400 (validation fails before any insert)" 400 POST /handle/batch-create \
  '[
    {"redirect":"https://www.example.com"},
    {"redirect":"ftp://bad-scheme.com"}
  ]'

run_curl "URL with embedded credentials → 400" 400 POST /handle/batch-create \
  '[{"redirect":"https://user:pass@example.com/"}]'

run_curl "invalid JSON body → 400" 400 POST /handle/batch-create \
  'not json'

run_curl "no auth → 401" 401 POST /handle/batch-create \
  '[{"redirect":"https://www.example.com"}]'

# Generate 1001-item payload (needs python3 or perl)
if command -v python3 &>/dev/null; then
  OVER_LIMIT=$(python3 -c "
import json
print(json.dumps([{'redirect': 'https://example.com/'} for _ in range(1001)]))")
  run_curl "1001 entries → 400 (exceeds limit)" 400 POST /handle/batch-create "$OVER_LIMIT"
else
  echo -e "${YELLOW}SKIP${NC}  1001-entry test — python3 not found"
  echo
fi

# ── batch-create → capture short IDs for batch-info tests ─────────────────────
separator "Setup: create URLs for batch-info tests"

echo "Creating 2 URLs to query in batch-info tests..."
CREATE_RESP=$(curl -s -X POST \
  "${AUTH_FLAGS[@]}" \
  -H "Content-Type: application/json" \
  -d '[
    {"redirect":"https://www.apple.com"},
    {"redirect":"https://www.microsoft.com"}
  ]' \
  "$HOST/handle/batch-create")

echo "$CREATE_RESP" | (command -v jq &>/dev/null && jq . || cat)
echo

if command -v jq &>/dev/null; then
  ID1=$(echo "$CREATE_RESP" | jq -r '.result[0].short // ""')
  ID2=$(echo "$CREATE_RESP" | jq -r '.result[1].short // ""')
else
  echo "jq not found — skipping batch-info tests that need created IDs"
  ID1=""
  ID2=""
fi

# ── batch-info ────────────────────────────────────────────────────────────────
separator "POST /handle/batch-info"

if [ -n "$ID1" ] && [ -n "$ID2" ]; then
  run_curl "2 existing IDs → 200 with redirect URLs and totals" 200 POST /handle/batch-info \
    "[\"$ID1\",\"$ID2\"]"

  run_curl "mix of existing and non-existent IDs → 200 (missing ones have empty redirect, total=0)" 200 POST /handle/batch-info \
    "[\"$ID1\",\"notexist999\"]"
else
  echo -e "${YELLOW}SKIP${NC}  existing-ID tests (no IDs captured)"
  echo
fi

run_curl "all non-existent IDs → 200 (redirect='', total=0 for each)" 200 POST /handle/batch-info \
  '["notexist1","notexist2","notexist3"]'

run_curl "empty array → 400" 400 POST /handle/batch-info \
  '[]'

run_curl "invalid JSON body → 400" 400 POST /handle/batch-info \
  'not json'

run_curl "no auth → 401" 401 POST /handle/batch-info \
  '["abc12"]'

# ── Summary ───────────────────────────────────────────────────────────────────
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "Result: ${GREEN}${PASS} passed${NC}  ${RED}${FAIL} failed${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
[ "$FAIL" -eq 0 ]
