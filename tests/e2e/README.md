# CLI flow coverage

Run `npm ci` and `npm run test:e2e` with Node 26 and the Go version in `go.mod`.
The locked runner is `tester-army/e2e` 0.17.0 (npm package `e2e`).
The engine-free CLI target needs no model, browser, API key, or production account.

Tests build the real executable once per suite realm. Each flow starts CLI subprocesses
with an isolated home, explicit config, synthetic credentials, and a local HTTP fixture.
The fixture checks requests at the network boundary. Tests inspect exit codes, output,
saved files, and subsequent commands. Tests do not import product logic or replace API helpers.
Each subprocess has a ten-second deadline. Transfer and HTTP deadline cases use shorter CLI deadlines.

## Controlled coverage matrix

All rows below execute against local fixtures. The catalog test checks the complete family and subcommand inventory.
Read cases assert returned data, exact paths, GET-only behavior, and relevant filters.
Mutation cases assert methods and payloads, then prove readonly and global dry-run send no writes.
The login and local-write flows test both flags across fourteen refusals.
Each refusal checks zero network requests, byte-identical config, and no new files beneath the isolated home.

| Family | Executed flows and observable outcomes |
| --- | --- |
| commands, version | JSON catalog, empty invocation, aliases/default reads, version text/JSON, literal flag delimiter |
| auth | CSRF and cookie login, password stdin, explicit and automatic refresh, concurrent refresh lock, token/env precedence, set-token, synthetic MMKV import, status, rejected auth and absent CSRF, config preservation, no token disclosure |
| config | set/get/unset, masked secrets, external editor persistence, frame selection, invalid keys/config, file secret encryption and reload, safe-read migration refusal |
| frames | list/show, devices/detail, alarms, household config, event/task notifications, reviews, reminders, nudges with time bounds, avatars/colors, set-default |
| categories | profile data in JSON/plain/text, missing frame, report dependencies |
| chores | list filters, search limit/lookback, week, streak, create/up-for-grabs, update, claim, complete, skip, delete, stdin bulk with partial validation failure |
| rewards | list, points, create/update/delete, redeem/unredeem, file bulk with partial failure |
| calendar | date filters, search, countdowns, sources, invites, timezone-aware week, create/countdown/update/delete |
| lists | list/detail, create/update/delete, item add/update/delete, completed-only clear and follow-up read, organize/order, task-box reads/writes |
| grocery | list/show/create, multiple-item add, empty clear after completed removal, organize/order, recipe add |
| meals | categories/recipes/detail/sittings, recipe create/update/delete, sitting create/delete, grocery add, portable recipe-ID mapping |
| photos | encoded cursor, detail, likes, comments page, signed upload and streamed bytes, downloads, transfer failure preserves file, delete, safety guards |
| albums | list, explicit message page, complete message IDs, bad page |
| routines | list/create/update/delete/reorder, required IDs |
| sidekick | sanitized Plus status and history, subscriptions and failure output |
| bounties | no inferred links in list, chore/reward create/update/delete, failed reward rollback, partial update receipt |
| rotations | two-week schedule and round-robin assignment, partial failure count, no-write dry-run |
| status, analytics, home | combined resource results, points/profile identities, date window, task exclusion, HTTP failure propagation |
| watch | once, persisted redeemed IDs, next-process resume, changed reward emission, SIGINT stop, poll failure |
| export, import | all supported resources, restrictive output mode, reference validation, both dry-run placements/counts, recipe remap, cross-frame rejection, failed export preserves existing file |
| raw | query encoding, stdin PATCH, large integer precision, off-origin credential omission, cross-origin redirect refusal, invalid body/trailing args, timeout and refused connection |
| safety/output | flag placement, readonly/env allowlist/denylist, mutation no-write proofs, expired-token and local-write refusal, TSV/text/JSON, conflicting formats, usage/runtime exits, 14 family HTTP-error JSON contracts |

## Evidence and limits

The suite reports JSON, JUnit, and Markdown under ignored `.e2e/`.
Synthetic fixtures verify executable contracts. They do not prove the current private API schema or feature entitlements.
Pagination tests verify explicit pages/cursors and complete-ID lookup. The CLI does not auto-follow pages.

`make live-readonly-smoke` is a separate real-account GET check. It requires a valid configured token.
Safety flags never refresh production credentials. An expired token prevents that live check.
Do not keep real account payloads in test artifacts or this repository.

The controlled suite does not drive a TTY password prompt, the macOS Keychain, or real Skylight MMKV files.
It tests password stdin, config storage, encrypted file storage, and synthetic MMKV parsing instead.
It does not send real email sessions, upload media to a real account, follow retailer checkout links,
or test remote mutations. Windows native Go checks remain in CI; these subprocess flows run on Linux.
