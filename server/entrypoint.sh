#!/usr/bin/env sh
set -e

mkdir -p db
goose -dir migrations sqlite3 db/game.db up

exec ./lunar-tear --host "${LUNAR_HOST}" --http-port "${LUNAR_HTTP_PORT}" --grpc-port "${LUNAR_GRPC_PORT:-443}"
