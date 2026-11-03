# Protocol: np-trophy

OpenPak's independent PlayStation 3 trophy service. Owns native console trophy
routes, discovery and the private storage API. It keeps no database: trophies live in
the network-wide [achievements](https://github.com/openpak/achievements) service (gRPC), where a PS3
communication id is title `NPWR...` on network `np`, a trophy is the zero-based index as a string,
and the same account sees one history across PS3, PS4 and PS5. Ticket validation rechecks
current account/link ownership through the account core, including interaction
policy when reading another user's trophies. np-account forwards the existing
native paths here; no account or XMPP implementation runs in this process.

Two separate listeners:

- `NP_HTTP_ADDR=:8531`: native `/trophy/sec/*`,
  `/np/resource/trophy/trophy-status.xml`, and `/healthz`.
- `NP_TROPHY_ADDR=:8530`: existing bearer-authenticated private
  `/v1/users/{core-uuid}/trophies` API and `/healthz`. Never publish this listener
  through console routing.

## Semantics

First-unlock semantics (a later sync never moves a recorded time), grade filling (a grade only
fills in where none was recorded) and the private API contract are unchanged; the achievements
service now enforces the first two. Grades travel as `B S G P` on this wire and as
`bronze/silver/gold/platinum` in the core. Reading lists every title the account has unlocks in
that is mapped on network `np`, under its lowest PS3 communication id. The service key stays on
`NP_TROPHY_KEY`; the achievements key is `NP_ACHIEVEMENTS_KEY`.
