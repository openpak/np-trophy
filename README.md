# np-trophy

OpenPak's PS3 trophy storage. A private API behind
[np-account](https://github.com/openpak/np-account): np-account verifies the firmware's
signed ticket, maps it to the OpenPak core account and calls this service with a bearer
key. np-trophy never faces a console and is never published outside the server network.

- `GET /healthz` — no auth.
- `GET /v1/users/{core-uuid}/trophies` — every stored unlock: `{"games": [{"communication_id",
  "trophies": [{"id", "earned_at", "grade"}]}]}`, by communication ID and trophy ID.
- `POST` the same path with `{"communication_id": "NPWR01234_00", "trophies": [...]}` — merges,
  then answers like `GET`. The first stored unlock of a trophy is kept (a later sync never moves
  its time); a grade (`B`, `S`, `G`, `P`) only fills in where none was recorded.

Bounds: canonical lower-case core UUIDs, 64 KiB body, 8 KiB headers, 1024 trophies per merge,
trophy IDs 0-1023, 5 s timeouts.

Configuration: `NP_TROPHY_DATABASE_URL` (Postgres, required), `NP_TROPHY_KEY` (at least
32 bytes, shared with np-account's `NP_TROPHY_BACKEND_KEY`), `NP_TROPHY_ADDR` (default `:8530`).

Test against an empty scratch database:

```sh
NP_TROPHY_TEST_DATABASE_URL=postgres://... go test ./...
```

Replaces the earlier RPCN-fork trophy facade with the same HTTP contract. Tag `v*.*.*` publishes
`ghcr.io/openpak/np-trophy` for amd64 and arm64. AGPL-3.0-only.
