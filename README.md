# 🎡 Wheel of Fortune

Build a wheel, share it, spin it. Every spin is settled **atomically in Redis**
by a Lua script, so the odds the UI shows are the odds you get, and limited
prizes can never be oversold, however many people spin at once.

![Home](docs/screenshots/home.png)

| Spin | Builder | Mobile (light) |
| --- | --- | --- |
| ![Spin](docs/screenshots/spin.png) | ![Builder](docs/screenshots/builder.png) | <img src="docs/screenshots/phone.png" alt="Mobile" width="220"> |

## Features

- **Three ways to pick a winner**
  | Mode | Chance of a segment winning | Good for |
  | --- | --- | --- |
  | **Equal** | `1 / n` over the segments that still have prizes | Lunch pickers, yes/no |
  | **Weighted** | `weight / Σ weights` over available segments | Loot tables, "1 in 100" jackpots |
  | **Inventory** | `remaining / Σ remaining` | Raffles that should drain evenly |
- **Limited or unlimited stock per segment.** Out-of-stock segments drop out
  of the draw automatically. Owners can pause, resume, restock or delete.
- **Firebase Auth** with Google, email/password or guest sign-in. Local
  development runs fully on the **Auth emulator**; no Firebase project needed.
- **Live odds, win counts and a recent-spins feed** on every wheel.
- **One port, every protocol.** The Go server speaks Connect, gRPC and gRPC-Web
  directly, so there is no Envoy proxy any more.
- **`wheelctl`**, a CLI to create, spin (load test with concurrency), inspect
  and manage wheels.

## Architecture

```
 browser (Vue 3 + Vuetify, Vite)          wheelctl / grpcurl / any gRPC client
        │  Connect over HTTP + Firebase ID token        │  gRPC or Connect
        └──────────────────────┬─────────────────────────┘
                               ▼
                    cmd/server  :7755  (Go, connect-go)
                    ├─ internal/auth        verify Firebase ID tokens
                    ├─ internal/server      proto <-> domain, error codes
                    ├─ internal/wheel       rules, validation, ownership
                    └─ internal/wheel/redisstore
                               │  EVALSHA spin.lua (atomic pick + decrement)
                               ▼
                             Redis
  Firebase Auth emulator :9099 (UI :4000) ◄── browser sign-in, server token checks
```

| Path | What |
| --- | --- |
| `proto/wheel/v1/wheel.proto` | API contract; the single source of truth |
| `gen/`, `web/src/gen/` | Generated code (`make gen`); do not edit |
| `cmd/server` | API server |
| `cmd/wheelctl` | CLI client |
| `internal/wheel` | Domain types, odds rules (`Wheel.Weights`), `Service` |
| `internal/wheel/redisstore` | Redis store and `spin.lua` |
| `internal/auth` | Firebase verifier and Connect interceptor |
| `web/` | Frontend ([web/README.md](web/README.md)) |

### Redis layout

Keys are hash-tagged on the wheel id so a wheel's keys share a cluster slot.

| Key | Type | Contents |
| --- | --- | --- |
| `wheel:{id}:def` | string | JSON definition: name, mode, segments, owner |
| `wheel:{id}:state` | hash | `enabled` (0/1), `spins` |
| `wheel:{id}:stock` | hash | segment index → remaining prizes (limited segments only) |
| `wheel:{id}:wins` | hash | segment index → wins |
| `wheel:{id}:history` | list | last 100 spins as JSON, newest first |
| `wheels` | zset | wheel ids scored by creation time |

A spin runs [`spin.lua`](internal/wheel/redisstore/spin.lua), which reads the
definition and stock, computes the weights for the wheel's mode, picks a
segment from a random value supplied by the server, then decrements stock,
increments wins and appends the history entry, all in one atomic script.

## Running locally

Prerequisites: [mise](https://mise.jdx.dev) (installs Go 1.27, Node 24, buf
and golangci-lint from `mise.toml`) and Docker. Without mise, install those
versions yourself.

```shell
make setup   # once: toolchain + web dependencies
make dev     # everything, in one go
```

`make dev` starts Redis and the Firebase Auth emulator in Docker (detached),
waits for them to be healthy, builds and runs the API, seeds three demo wheels
if Redis is empty, and runs the Vite dev server. Logs from both are shown with
`[api]` / `[web]` prefixes. Open **http://localhost:5173** and choose
*Play as a guest*.

- **Ctrl-C** stops the API and web. The containers keep running, so the next
  `make dev` starts instantly; `make down` stops them (and discards wheels).
- If port 6379 is taken (e.g. by another Redis), it picks the next free port.
  Set `REDIS_PORT` to choose one, or `SEED=0` to skip the demo wheels.
- Also running: the emulator UI on http://localhost:4000 (test users) and
  RedisInsight on http://localhost:8001 (browse the `wheel:{id}:*` keys).

Prefer separate terminals? `make deps`, `make server`, `make web` and
`make seed` do the individual steps.

To run the API in Docker as well: `docker compose --profile server up --build`.

### wheelctl

With `FIREBASE_AUTH_EMULATOR_HOST` set, `wheelctl` signs in to the emulator as
`cli@wheeloffortune.dev` (pass `-email` to act as someone else). Otherwise pass
a Firebase ID token with `-token` or `WHEEL_TOKEN`.

```shell
export FIREBASE_AUTH_EMULATOR_HOST=127.0.0.1:9099
alias wheelctl='go run ./cmd/wheelctl'

# segments are label[:weight[:stock]]; an empty stock means unlimited
wheelctl create -name "Loot box" -mode weighted -s Common:60 -s Rare:30 -s Epic:9 -s Legendary:1
wheelctl create -name "Raffle" -mode inventory -s "Headphones::1" -s "Mug::10"
wheelctl list
wheelctl spin -n 1000 -c 16 loot-box-abc123   # prints the observed distribution
wheelctl get loot-box-abc123
wheelctl history raffle-xyz789
wheelctl pause|resume|restock|delete <id>
```

The server also serves gRPC reflection and health checks, so
`grpcurl -plaintext localhost:7755 list` works.

### Configuration

| Flag | Env | Default |
| --- | --- | --- |
| `-addr` | `ADDR` | `:7755` |
| `-redis-url` | `REDIS_URL` | `redis://localhost:6379/0` |
| `-auth` | `AUTH_MODE` | `firebase` (`none`: every caller is one local user) |
| `-firebase-project` | `FIREBASE_PROJECT_ID` | `demo-wheeloffortune` |
| | `FIREBASE_AUTH_EMULATOR_HOST` | unset (set it to use the emulator) |
| `-cors-origins` | `CORS_ORIGINS` | `http://localhost:5173,http://127.0.0.1:5173` |
| `-log-level` | `LOG_LEVEL` | `info` |
| `-log-json` | `LOG_FORMAT=json` | text |

Frontend settings live in [`web/.env.example`](web/.env.example).

### Using real Firebase

1. Create a Firebase project and enable the Google, Email/Password and
   Anonymous providers.
2. Server: set `FIREBASE_PROJECT_ID` and leave `FIREBASE_AUTH_EMULATOR_HOST`
   unset. Verifying tokens only needs the project id.
3. Web: set `VITE_FIREBASE_API_KEY` and `VITE_FIREBASE_PROJECT_ID`, and set
   `VITE_FIREBASE_AUTH_EMULATOR_URL=` (empty).

## Development

```shell
make gen    # after editing proto/: buf lint + regenerate Go and TS
make test   # go test -race ./... and vitest
make lint   # buf lint, go vet, golangci-lint, vue-tsc
```

Store tests use in-process [miniredis](https://github.com/alicebob/miniredis)
by default. Set `TEST_REDIS_URL=redis://localhost:6379/15` to run them against
real Redis (this **flushes that DB**). CI does both lint and real-Redis tests,
and fails if generated code is stale.

## API

See [`proto/wheel/v1/wheel.proto`](proto/wheel/v1/wheel.proto). In short:

| RPC | Auth |
| --- | --- |
| `GetWheel`, `ListWheels`, `ListSpins` | public |
| `CreateWheel`, `SpinWheel` | any signed-in user (guests included) |
| `SetWheelEnabled`, `RestockWheel`, `DeleteWheel` | wheel owner |

Errors use standard codes: `invalid_argument` (validation), `not_found`,
`unauthenticated`, `permission_denied`, `failed_precondition` (paused or out of
prizes).

## License

[Apache 2.0](LICENSE.md)
