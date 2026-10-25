# np-trophy

OpenPak's independent PlayStation 3 trophy service. Owns native console trophy
routes, discovery and persistent Postgres storage. Ticket validation rechecks
current account/link ownership through the account core, including interaction
policy when reading another user's trophies. np-account forwards the existing
native paths here; no account or XMPP implementation runs in this process.

Two separate listeners:

- `NP_HTTP_ADDR=:8531`: native `/trophy/sec/*`,
  `/np/resource/trophy/trophy-status.xml`, and `/healthz`.
- `NP_TROPHY_ADDR=:8530`: existing bearer-authenticated private
  `/v1/users/{core-uuid}/trophies` API and `/healthz`. Never publish this listener
  through console routing.

Required: `NP_TROPHY_DATABASE_URL`, `NP_TROPHY_KEY` (at least 32 bytes),
`NP_CORE_KEY` (at least 32 bytes), `NP_TICKET_KEY` (existing Ed25519 PKCS8 key),
`NP_TROPHY_RESOURCE_KEY` (existing 20-byte firmware compatibility key).
`NP_CORE_ADDR` defaults to `localhost:7000`. Optional `NP_TLS_ADDR`,
`NP_TLS_CA_CERT`, `NP_TLS_CA_KEY` enable the firmware TLS front on native HTTP.
No OAuth credentials or `NP_TOKEN_KEY` are needed.

The native handlers use local storage directly. Existing schema, first-unlock
semantics, grade filling and private API contract remain unchanged. Reuse the
existing database; no export/import or new identity database is needed.
Discovery advertises maintenance on storage failure.

```sh
# Export values from example.env after supplying operator credentials.
GOWORK=off go run -mod=vendor .
GOWORK=off go test -mod=vendor ./...
GOWORK=off go vet -mod=vendor ./...
podman build -t np-trophy .
# Runs database integration tests against an EMPTY scratch database:
NP_TROPHY_TEST_DATABASE_URL=postgres://... GOWORK=off go test -mod=vendor ./...
```

Shared np-common code is vendored; refresh using `GOWORK=off go mod vendor`
with the sibling module present. Release tags publish `ghcr.io/openpak/np-trophy`.
See [migration](https://github.com/openpak/np-account/blob/main/docs/service-split.md)
and [wire provenance](docs/provenance.md). AGPL-3.0-only.
