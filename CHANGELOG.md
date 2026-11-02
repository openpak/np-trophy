# Changelog: np-trophy

## v0.2.1 — 2026-11-02

- Uses github.com/openpak/account-api instead of a vendored accountpb copy; docs trimmed to README, protocol and deploy.

## v0.2.0 — 2026-10-25

- Split PS3 account, community/XMPP and native trophies into independent services; share protocol primitives through vendored np-common.


## v0.1.0 — 2026-10-16

- First release: Postgres-backed PS3 trophy storage behind np-account, same HTTP contract
  as the RPCN-fork facade it replaces (bearer key, first unlock kept, grade fills once).
