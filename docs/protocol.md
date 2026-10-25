# Protocol: np-trophy

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
