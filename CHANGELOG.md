# Changelog: np-trophy

## v0.3.1 — 2026-11-03

- Every request logs one stderr line, `http <method> <path> -> <status> <duration>ms` (np-common v0.1.2 `httplog`,
  through `nphttp.Logging`): no query, headers, credentials or bodies; `/healthz` skipped; the PS3 status header is
  appended as `ps3_status=<n>` when set. Replaces the slog "NP HTTP" line, which had no duration.

## v0.3.0 — 2026-11-03

- Trophies are stored in the neutral achievements core (board #59): `store.go` and the Postgres
  dependency are gone, replaced by a gRPC client (`internal/achievementspb`, `NP_ACHIEVEMENTS_ADDR`
  and `NP_ACHIEVEMENTS_KEY`). The PS3 wire, the private API and the first-recorded-unlock-wins
  rule are unchanged; the core enforces the rule. `NP_TROPHY_DATABASE_URL` is no longer read.

## v0.2.1 — 2026-11-02

- Uses github.com/openpak/account-api instead of a vendored accountpb copy; docs trimmed to README, protocol and deploy.

## v0.2.0 — 2026-10-25

- Split PS3 account, community/XMPP and native trophies into independent services; share protocol primitives through vendored np-common.


## v0.1.0 — 2026-10-16

- First release: Postgres-backed PS3 trophy storage behind np-account, same HTTP contract
  as the RPCN-fork facade it replaces (bearer key, first unlock kept, grade fills once).
