#!/bin/sh
set -euo pipefail

cd "$(dirname "$0")"


if [ -f .env ]; then
  export $(grep -E '^[A-Z_][A-Z0-9_]*=' .env | xargs)
fi

exec ./go_pg_dump
