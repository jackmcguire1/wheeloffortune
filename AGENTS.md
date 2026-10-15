# AGENTS.md

Guidance for AI coding agents (and humans) working in this repo. See
README.md for the product overview.

## Layout

- `proto/wheel/v1/wheel.proto` is the API contract. Change it first, then run
  `make gen`. Never hand-edit `gen/` or `web/src/gen/`; CI fails if they are stale.
- `cmd/server` is the Connect/gRPC/gRPC-Web server. `cmd/wheelctl` is the CLI.
- `internal/wheel` holds the domain: types, validation, the `Service`
  (ownership and auth checks) and the odds rules in `Wheel.Weights`.
- `internal/wheel/redisstore` implements `wheel.Store` on Redis, with `spin.lua`.
- `internal/auth` verifies Firebase ID tokens and has the Connect interceptor
  that puts the `*wheel.User` on the context (`auth.UserFrom`).
- `internal/server` maps proto to domain and domain errors to Connect codes.
- `web/` is Vue 3 (`<script setup lang="ts">`), Vuetify 4, Pinia, vue-router,
  Vite and connect-web.

## Invariants: keep these in sync

The odds rules exist in **three** places. Change all three together and update
the tests:

1. `internal/wheel/wheel.go`: `Wheel.Weights` (what the API reports)
2. `internal/wheel/redisstore/spin.lua` (what actually decides a spin)
3. `web/src/lib/probability.ts` (builder preview and slice layout)

Validation limits (`MaxSegments`, `MaxLabelLen`, `MaxWeight`, `MaxStock`, …)
in `internal/wheel/wheel.go` are mirrored in `web/src/pages/BuilderPage.vue`.

Other rules:
- Spins must stay atomic: pick, decrement stock, count the win and record
  history in one Lua script. Don't split them into separate Redis calls.
- The random value comes from Go (`Service.rand`, injectable with
  `wheel.WithRand` for tests) and is passed to Lua as `ARGV[1]`.
- Segment indexes are zero based everywhere outside Lua.
- Auth: reads are public, create/spin need a user, and pause/restock/delete
  need the owner. Enforce this in `wheel.Service`, not in the handlers.
- Return domain errors (`wheel.ErrNotFound`, `*wheel.ValidationError`, …) from
  the domain and map them only in `internal/server`.

## Commands

```shell
make dev      # all of the below in one command (scripts/dev.sh)
make deps     # Redis + Firebase Auth emulator in Docker
make server   # API on :7755 against the emulator
make web      # Vite on :5173
make gen      # buf lint + generate Go/TS
make test     # go test -race ./... && vitest
make lint     # buf lint, go vet, golangci-lint, vue-tsc
```

Toolchain versions are pinned in `mise.toml` (Go 1.27.2, Node 24, buf,
golangci-lint). Keep them on the latest stable releases and bump
golangci-lint together with Go.

## Testing

- Domain tests: `internal/wheel/*_test.go`.
- Store tests run on miniredis by default. Set
  `TEST_REDIS_URL=redis://localhost:6379/15` for real Redis (it flushes that DB).
  Changes to `spin.lua` should be checked against real Redis.
- `internal/server/server_test.go` drives the full Connect stack over
  `httptest` with a fake token verifier. Add a case there for new RPCs.
- Frontend: `cd web && npm test` (vitest) and `npm run typecheck`.
- End to end: `make dev`, then use `bin/wheelctl` with
  `FIREBASE_AUTH_EMULATOR_HOST=127.0.0.1:9099`, or the browser with
  "Play as a guest".

## Style

- Go: standard library first (`log/slog`, `net/http`, `flag`), wrap errors
  with `%w`, accept a `context.Context` first, and keep comments brief.
  `golangci-lint run` must be clean.
- Frontend: Composition API with `<script setup lang="ts">`, Vuetify
  components over custom CSS, and icons from `@mdi/js`. Make sure
  `npm run build` passes.
- Commits: conventional prefixes (`feat:`, `fix:`, `docs:`, `chore:`).
