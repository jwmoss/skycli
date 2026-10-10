# API capabilities and gaps

Audit date: 2026-10-10.
The API is private. Skylight does not publish a verified complete REST specification.
Command counts do not measure feature coverage.

This audit compares the [official feature releases](https://skylight.zendesk.com/hc/en-us/categories/42813896436763-What-s-New),
Skylight 2.26.0 call sites, and live account reads.
The App Store lists 2.27.0; that binary is not inspected.
The [contract audit](api-contracts-2026-10.md) records the bundle hash, methods, payloads, and evidence limits.
The initial audit used local request fixtures. The retained suite now uses simple live end-to-end requests.

## Coverage

Dedicated commands expose the following controls. Some advanced fields use `--body` or `--body-file`.
These JSON options send fields through the verified route; they do not establish an undocumented field's meaning.

| Area | Supported controls | Remaining gaps |
|---|---|---|
| Profiles and labels | List/show/create/update/delete; conversion; delete with reassignment; avatar/hat IDs through JSON; hat catalogue | Custom profile-image uploads; birthday and buddy appearance commands |
| Chores and linked tasks | Create/update/delete; multiple profiles; due time; recurrence and completion-based repeats; completion/skip/undo/unskip; ordering; occurrence and profile scopes | Dedicated timer controls; complete history backup |
| Routines and habits | Actual chore-backed routines; time-of-day, recurrence, habit flag, profiles, completion/skip/undo/unskip, ordering | Separate upstream habit-score report; nested step-list format is not an app contract |
| Calendar events | Range/search/week/countdowns; create/update/delete; multiple profiles; recurrence; invitations; location/description; sync destination; edit scopes | Automatic preservation of recurrence exceptions when replacing the set; presentation views/widgets |
| Synced calendars | List; create/update/delete source attributes; default source; profile mappings; active-calendar selection | Provider OAuth connection/reconnection and consent; provider-specific setup schemas |
| Rewards | List/create/update/delete/redeem/unredeem; balances; direct signed point adjustments | Complete points/redemption history backup |
| Lists and grocery | Lists/items CRUD; organization; grocery order initiation; recipe ingredients | Dedicated list-section controls; full pagination audit; external checkout completion |
| Task Box | Read/create/update/delete saved task templates; reuse fields with normal task creation | Dedicated convenience command to assign a saved template |
| Meals and recipes | Recipe CRUD; meal-category update; sitting create/update/reschedule/delete; recurrence; grocery transfer | Complete recipe search and recipe-deletion scope audit |
| Photos and albums | Upload/download; photo details; captions, likes, comments, cross-frame copies; bulk deletion; album CRUD and membership | Automatic full pagination; video processing and new multipart/batch upload flows |
| Sidekick | Access/history; JSON intent creation; results; typed draft reads/approval; undo | Photo/PDF/file uploads; editing unapproved drafts; full engine-specific input builders |
| Device and household settings | Device update/sleep/wake; household update; device-config read/update; event/task notification update | Alarm writes; explicit flag for October announcement frequency; licensed-feature behavior |
| Nudges and household access | Nudge list/create/update/delete; users; reminder profile and month reviews | Parent-voice audio upload; reminder-profile edits; invite links and access management |

Routines now use the chores API with `routine=true`.
The old routines endpoint returned HTTP 404 during the baseline smoke test.
Profile/task JSON preserves new attributes and relationships.
Calendar week JSON preserves metadata while it converts timestamp offsets.

## Latest official features

The [October 7 announcement-frequency control](https://skylight.zendesk.com/hc/en-us/articles/56891404396571--Feature-Choose-How-Often-New-Features-Are-Announced)
is the newest entry checked. The device-config route is known; its specific field remains unverified.
The CLI does not claim support through an invented JSON key.

The audit covers recent [profile deletion](https://skylight.zendesk.com/hc/en-us/articles/55629592365083--Feature-Delete-Profiles-and-Labels),
[profile hats](https://skylight.zendesk.com/hc/en-us/articles/55019080026139--Feature-Profile-Hats),
[linked tasks](https://skylight.zendesk.com/hc/en-us/articles/55207742218267--Feature-Linked-Tasks),
[task-completion notifications](https://skylight.zendesk.com/hc/en-us/articles/54930439904923--Feature-Task-Completion-Notifications),
and [completion-based repeats](https://skylight.zendesk.com/hc/en-us/articles/53932501748251--Feature-Repeat-a-Chore-After-Completion).
[Magic Import draft edits](https://skylight.zendesk.com/hc/en-us/articles/55093100299931--Feature-Edit-Drafts-of-Magic-Imports)
remain a gap because the full write chain is not established.

Official help defines product features, not wire contracts.
[Calendar Plus](https://skylight.zendesk.com/hc/en-us/articles/32171114576283-What-is-Calendar-Plus)
and household permissions can restrict availability.
UI gestures, display layouts, and widgets do not imply separate CLI endpoints.

## Verification limits

`make ci` runs build and static checks without an account.
`make test` runs real GET requests for categories, chores, routines, calendar events, and lists.
It then creates one hidden temporary list, reads it back, deletes it, and checks cleanup.
`make live-readonly-smoke` runs only the GET checks.
The suite uses the built CLI and its configured account and frame. It has no fake server or unit tests.
HTTP, transport, and JSON failures fail the run.
This small suite does not establish every command's behavior.
The October 10 live run passes all five GET checks and the temporary-list create/read/delete cycle.

The earlier October 10 read-only audit passes with a temporary `2026-08-05` config.
Device configuration, household users, hats, and individual profiles pass those live GET checks.
The routine read succeeds but returns zero instances for the selected date.
The account has no albums or Sidekick intents, so their detail/draft checks remain unverified.

The API default now matches the inspected app: `2026-08-05`.
Existing `api_version` config values remain explicit overrides.
Write behavior beyond the temporary-list cycle, device behavior, and full 2.27.0 parity remain unverified.

## Export and local calculations

[Export](commands/export.md) produces selected templates, not a complete account backup.
[Import](commands/import.md) validates target references and maps new recipe IDs.
It rejects cross-frame files and can create duplicates on repeated runs.

`chores streak` calculates an assignee's completion across all recorded chores.
It excludes skipped chores and does not reproduce Skylight's per-routine habit score.
Bounty commands use explicit IDs; list output does not infer links from matching point values.

## Further parity work

Inspect the latest app before adding version-specific fields.
Capture authorized, sanitized write evidence for the remaining workflows.
Keep method, route, fields, subscription, role, and verification date with each contract.
Use the Skylight app for provider consent and unsupported uploads.
The `raw` command is an escape hatch, not evidence of complete feature support.
