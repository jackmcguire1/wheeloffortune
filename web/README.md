# Wheel of Fortune web

Vue 3 + Vuetify 4 + Vite frontend for the wheel server. It talks to the Go
server over [Connect](https://connectrpc.com) and signs users in with Firebase
Auth (the local emulator by default).

```shell
cp .env.example .env.local   # optional, the defaults match local dev
npm install
npm run dev                  # http://localhost:5173
```

The server (`:7755`) and the Firebase Auth emulator (`:9099`) need to be
running; see the root README.

| Script              | What it does                         |
| ------------------- | ------------------------------------ |
| `npm run dev`       | Dev server with HMR                  |
| `npm run build`     | Type-check and build to `dist/`      |
| `npm run typecheck` | `vue-tsc` only                       |
| `npm test`          | Vitest unit tests                    |

`src/gen` is generated from `proto/` by `buf generate` at the repo root. Don't
edit it by hand.

`src/lib/probability.ts` mirrors the server's odds rules so the builder preview
and the slice layout match what the server decides.
