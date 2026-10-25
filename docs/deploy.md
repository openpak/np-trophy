# Deploy: np-trophy

Required: `NP_TROPHY_DATABASE_URL`, `NP_TROPHY_KEY` (at least 32 bytes),
`NP_CORE_KEY` (at least 32 bytes), `NP_TICKET_KEY` (existing Ed25519 PKCS8 key),
`NP_TROPHY_RESOURCE_KEY` (existing 20-byte firmware compatibility key).
`NP_CORE_ADDR` defaults to `localhost:7000`. Optional `NP_TLS_ADDR`,
`NP_TLS_CA_CERT`, `NP_TLS_CA_KEY` enable the firmware TLS front on native HTTP.
No OAuth credentials or `NP_TOKEN_KEY` are needed.

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
and [wire provenance](provenance.md). AGPL-3.0-only.
