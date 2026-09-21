# API capabilities and gaps

This ledger records the command surface checked on 2026-09-20.
The API is private. There is no verified, complete upstream REST specification.
A command count is not an API coverage percentage.

Dedicated commands use known routes. Some preserve raw JSON or accept extra fields through `--body`.
The `raw` command sends a supplied method, path, query, and JSON body.
It does not discover routes, validate undocumented fields, or prove account access.

## Coverage ledger

GET means a read. POST, PUT, PATCH, and DELETE can change remote data.
This table describes source coverage and offline request tests. It does not claim current live-account verification.

| Area | Dedicated reads | Dedicated writes | Export scope | Known gaps |
|---|---|---|---|---|
| Auth/account | Local auth status; GET user through doctor | OAuth login/refresh; local token storage | None | Account management and household access APIs |
| Frames/devices | GET frames, devices, alarms, household config, avatars, colors | Local default frame only | None | Remote device, alarm, and settings changes |
| Profiles/categories | GET categories | None | References only | Create, edit, merge, and calendar-to-profile mapping |
| Chores | GET list/search; local week/streak | POST create; PUT update, claim, complete, skip; DELETE | Selected definitions | Dedicated undo/unskip, linked multiple profiles, completion-based repeats, due-time controls |
| Routines | GET routines | POST create; PUT update; DELETE; PATCH reorder | None | Habit/time-of-day flags and routine completion/skip controls |
| Rewards | GET rewards and points | POST create/redeem/unredeem; PATCH update; DELETE | Definitions only | Points/history backup |
| Calendar events | GET range/search/countdowns/recent invites; local week | POST create/countdown; PUT update; DELETE | Selected fields | Dedicated recurrence, invitations, multiple profiles, and sync destination controls |
| Synced calendars | GET source calendars | None | None | Connect, reconnect, remove, and select default source |
| Lists/grocery | GET lists and items | POST/PUT/DELETE lists/items; POST organize/order/add recipe | Lists and items | Verified full pagination coverage; external checkout completion |
| Task Box | GET items | POST item | None | Update, delete, search, and apply-to-profile controls |
| Recipes | GET categories, recipes, and one recipe | POST/PATCH/DELETE recipe | Selected fields | Category changes and additional recipe fields |
| Meal sittings | GET date range | POST sitting; DELETE dated instance | Selected fields | Update, reschedule, and recurrence controls |
| Photos | GET page/detail/likes/comments; asset download | POST upload URL; asset PUT; DELETE multiple | None | Automatic full pagination, like/comment writes, existing caption edits, cross-household copy |
| Albums | GET albums/messages/all message IDs | None | None | Create, rename, delete, and add/remove photos |
| Sidekick | GET Plus access and auto-creation history | None | None | Event/list/recipe imports and meal generation |
| Notifications/reviews/nudges | GET settings, reviews, reminder profile, nudges | None | None | Settings changes and nudge triggers |
| Household access | No dedicated command | None | None | Invite links and access management |

Implementation sources: [client](../internal/skylight/client.go), [calendar](../internal/skylight/calendar.go),
[lists](../internal/skylight/lists.go), [meals](../internal/skylight/meals.go),
[routines](../internal/skylight/routines.go), [photos](../internal/skylight/photos.go),
[albums](../internal/skylight/albums.go), [Sidekick](../internal/skylight/sidekick.go),
[frame resources](../internal/skylight/frame_resources.go), and [other reads](../internal/skylight/features.go).

## Evidence and access limits

Official help confirms these user-facing features:

- [Profiles](https://roadmap.ourskylight.com/Profiles-d3b20bb1fa9c4e709c4058c1145db175) can map calendars to household members.
- [Tasks](https://skylight.zendesk.com/hc/en-us/articles/44738601403931-Tasks-Routines-and-Chores) support linked profiles, due times, completion-based repeats, undo, and unskip.
- [Habit Tracker](https://skylight.zendesk.com/hc/en-us/articles/50672917447067-Track-Your-Habits) preserves per-routine streaks across skips.
- [Photos](https://skylight.zendesk.com/hc/en-us/articles/44740038167067-Photos) supports album changes, captions, likes, and copies between households.
- [Sidekick](https://skylight.zendesk.com/hc/en-us/articles/39335273393947-Sidekick) supports content imports and meal generation.
- [Calendar Plus](https://skylight.zendesk.com/hc/en-us/articles/32171114576283-What-is-Calendar-Plus) gates Sidekick, meal planning, rewards, and screensaver features.
- [Calendar sharing](https://skylight.zendesk.com/hc/en-us/articles/32077029247643-Sharing-Access-To-Calendar) grants invite recipients access to manage calendar content.

These pages establish feature gaps. They do not define REST methods, routes, payloads, or precise role permissions.
No missing write route is guessed from a feature name.

Before adding a command, record an authorized, sanitized request/response fixture.
Record its method, route, fields, pagination, account role, subscription, and successful verification date.
Add a request-contract test and a read-only smoke check where applicable.
Treat HTTP 403/404 as account/endpoint unavailability only in explicitly optional smoke checks.
Other HTTP, transport, and invalid-JSON failures must fail the check.

## Export and local calculations

[Export](commands/export.md) produces selected templates, not a complete backup.
[Import](commands/import.md) validates target references and maps new recipe IDs.
It rejects cross-frame files and can create duplicates on repeated runs.

`chores streak` calculates an assignee's completion across all recorded chores.
It excludes skipped chores. It is not the upstream per-routine habit score.
Bounty commands use explicit IDs; list output no longer invents links from equal point values.

## Small next steps

Add only verified controls that a real workflow needs.
Useful candidates are profile selection, event sync destination, task undo/unskip, routine habit fields, and meal updates.

For school events, first use a native calendar feed.
[ClassReach documents its calendar feed](https://help.classreach.com/syncing-your-classreach-calendar-with-external-calendars-like-gmail-or-ical).
[Skylight supports one-way subscriptions](https://skylight.zendesk.com/hc/en-us/articles/41912303413659-Skylight-Calendar-One-Way-Sync-with-Apple-iCloud-Calendar)
and [native two-way iCloud sync](https://skylight.zendesk.com/hc/en-us/articles/41912365519131-Skylight-Calendar-Two-Way-Sync-with-Apple-iCloud-Calendar).
Keep private feed URLs out of logs and fixtures.
A custom sync service is useful only when the native feed cannot supply the required data.
