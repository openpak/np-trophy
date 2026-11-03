# PS3 Trophy Server

OpenPak's PS3 Trophy Server.

| | |
| --- | --- |
| Network | PlayStation 3 |
| Role | Native PS3 trophy routes, discovery and the private trophy API; trophies are stored in the achievements core |
| Depends on | [`account`](https://github.com/openpak/account) (account core), [`np-common`](https://github.com/openpak/np-common) (vendored), [`achievements`](https://github.com/openpak/achievements) (trophy storage, gRPC) |
| Used by | [`np-account`](https://github.com/openpak/np-account) |
| Image | `ghcr.io/openpak/np-trophy` |
| Licence | AGPL-3.0-only |

## What it serves

| Endpoint | What |
| --- | --- |
| `/trophy/sec/*`, `/np/resource/trophy/trophy-status.xml` | Native PS3 trophy routes and discovery (native HTTP listener, `:8531`) |
| `/v1/users/{core-uuid}/trophies` | Private bearer-authenticated trophy API (separate listener, `:8530`, never published to consoles) |
| `/healthz` | Health check on both listeners |

## Run

```sh
# Export values from example.env after supplying operator credentials.
GOWORK=off go run -mod=vendor .
```

## Docs

- [Protocol](docs/protocol.md)
- [Deploy](docs/deploy.md)
- [Wire provenance](docs/provenance.md)

## Licence

AGPL-3.0-only. See [LICENSE](LICENSE).
