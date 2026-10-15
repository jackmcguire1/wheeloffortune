#!/usr/bin/env bash
# One command local dev: Redis + Firebase Auth emulator in Docker (detached),
# then the API and the Vite dev server in the foreground. Ctrl-C stops the API
# and web; the containers keep running until `make down`.
#
#   SEED=0       skip creating demo wheels when Redis is empty
#   REDIS_PORT   force a host port for Redis (default: 6379, or the next free one)
set -euo pipefail
cd "$(dirname "$0")/.."

bold=$'\033[1m' dim=$'\033[2m' cyan=$'\033[36m' magenta=$'\033[35m' red=$'\033[31m' reset=$'\033[0m'
say() { printf '%s▸%s %s\n' "$cyan" "$reset" "$*"; }
die() { printf '%s✗ %s%s\n' "$red" "$*" "$reset" >&2; exit 1; }

port_busy() { lsof -nP -iTCP:"$1" -sTCP:LISTEN >/dev/null 2>&1; }

# Reuse the port of an already running compose service, otherwise find a free one.
pick_port() {
  local service=$1 container_port=$2 port=$3 published
  published=$(docker compose port "$service" "$container_port" 2>/dev/null | awk -F: 'NF {print $NF}')
  if [ -n "$published" ]; then
    echo "$published"
    return
  fi
  while port_busy "$port"; do port=$((port + 1)); done
  echo "$port"
}

command -v docker >/dev/null || die "docker is not installed"
command -v go >/dev/null || die "go is not installed (run: mise install)"
command -v npm >/dev/null || die "npm is not installed (run: mise install)"
docker info >/dev/null 2>&1 || die "docker is not running"

for port in 7755 5173; do
  if port_busy "$port"; then
    die "port $port is already in use ($(lsof -nP -iTCP:$port -sTCP:LISTEN | awk 'NR==2 {print $1, "pid", $2}')). Stop it and retry."
  fi
done

export REDIS_PORT=${REDIS_PORT:-$(pick_port redis 6379 6379)}
export REDISINSIGHT_PORT=${REDISINSIGHT_PORT:-$(pick_port redis 8001 8001)}
[ "$REDIS_PORT" = 6379 ] || say "port 6379 is taken, using ${bold}$REDIS_PORT${reset} for Redis"

say "starting Redis and the Firebase Auth emulator (detached)…"
docker compose up -d --build --wait redis firebase

if [ ! -d web/node_modules ]; then
  say "installing web dependencies…"
  (cd web && npm ci)
fi

say "building the API…"
go build -o bin/ ./cmd/server ./cmd/wheelctl

export REDIS_URL="redis://localhost:$REDIS_PORT/0"
export FIREBASE_AUTH_EMULATOR_HOST=127.0.0.1:9099

# Stop the API and web (only our own children) on exit or Ctrl-C.
cleanup() {
  trap - EXIT INT TERM
  pkill -TERM -P $$ 2>/dev/null || true
  wait 2>/dev/null || true
  printf '\n%s▸%s stopped API and web. Containers are still up: make down\n' "$cyan" "$reset"
  exit 0
}
trap cleanup EXIT INT TERM

prefix() { awk -v p="$1" '{ print p $0; fflush() }'; }

bin/server 2>&1 | prefix "${magenta}[api]${reset} " &
api=$!

for _ in $(seq 1 50); do
  curl -sf http://localhost:7755/healthz >/dev/null && break
  kill -0 "$api" 2>/dev/null || die "API failed to start"
  sleep 0.2
done
curl -sf http://localhost:7755/healthz >/dev/null || die "API did not become healthy"

if [ "${SEED:-1}" != 0 ] && [ -z "$(bin/wheelctl list | tail -n +2)" ]; then
  say "no wheels yet, creating demo wheels…"
  WHEELCTL=bin/wheelctl make -s seed >/dev/null
fi

# exec vite directly so it is our child and receives the stop signal.
(cd web && exec node_modules/.bin/vite --clearScreen false) 2>&1 | prefix "${cyan}[web]${reset} " &
web=$!

cat <<EOF

  ${bold}🎡 Wheel of Fortune is running${reset}

  App            ${bold}http://localhost:5173${reset}
  API            http://localhost:7755
  Emulator UI    http://localhost:4000
  RedisInsight   http://localhost:$REDISINSIGHT_PORT
  ${dim}CLI            FIREBASE_AUTH_EMULATOR_HOST=127.0.0.1:9099 bin/wheelctl list${reset}

  ${dim}Ctrl-C to stop · make down to stop the containers${reset}

EOF

# Exit (and clean up) as soon as either process dies.
while kill -0 "$api" 2>/dev/null && kill -0 "$web" 2>/dev/null; do sleep 1; done
