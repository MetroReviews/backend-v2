# Metro Reviews backend-v2 — Current Standing

_Last updated: 2026-08-17. This is a working status doc, not for publishing — it's here so you can look at one file and know what exists, how it fits together, and what's still open._

## TL;DR

This repo is the Go backend for Metro Reviews: a single service that runs both the **HTTP API** and the **Discord bot**, backed by one Postgres database. It compiles clean (`go build ./...` and `go vet ./...` both pass with nothing to report) and the domain logic reads as finished, not scaffolded. What's missing is the stuff around the edges: almost no automated tests, no developer-facing README, no CI, and a config file that ships with every secret blank.

The big recent shift, visible directly in the code: Metro used to be **only** a Discord bot list (old `silverpelt` package + `routes/lists`, both now deleted) that reviewed bots and then fanned the result out over webhooks to other bot lists. That's gone. It's been replaced by a **generic, category-based businesses platform** (`routes/businesses`, `routes/categories`) that reuses the exact same staff review queue for anything, not just bots — Discord bots are now just one category of business among others.

---

## Architecture, in one picture

```
                     ┌─────────────────────┐
                     │      main.go        │
                     │  wires everything up │
                     └──────────┬───────────┘
                                │
              ┌─────────────────┼──────────────────┐
              │                 │                  │
      ┌───────▼───────┐  ┌──────▼───────┐  ┌───────▼────────┐
      │   HTTP API     │  │ Discord bot   │  │   Postgres      │
      │  (chi router)  │  │ (discordgo)   │  │  (pgx pool)     │
      └───────┬────────┘  └──────┬───────┘  └───────┬────────┘
              │                  │                    │
              └────────┬─────────┴────────────────────┘
                        │
              shared domain packages:
              review/ · roles/ · perms/ · identity/
```

- **Entry point:** [main.go](main.go) — loads config, opens the DB pool (which auto-applies migrations), starts the Discord bot if a token is configured, then wires up every API router and starts listening.
- **One process, two interfaces.** The API and the bot aren't separate services — they're the same binary, sharing the same DB pool and the same domain packages (`review`, `roles`, `identity`), so a claim/approve/deny action does the same thing whether it comes from `/queue` in Discord or from a POST to `/businesses/{id}/review/approve`.
- **No panel/frontend code lives here.** This repo is API + bot only. `routes/panel` is just the backend half of the login handshake (OAuth2 URL + callback + a ticket-based access check) — the actual staff panel UI is a separate project that talks to these endpoints.

---

## The domain model

Two tables carry the actual "things being reviewed":

| Table | What it is |
|---|---|
| `bots` | Metro's own Discord bot list. Legacy in the sense that it predates the generic platform, but still fully first-class — same queue, same review commands. |
| `businesses` | The generic side: any service/business, tagged with a `category_id`, with address/city/country/website/logo/banner fields plus a free-form `metadata` JSON blob for category-specific extras. |

Everything else hangs off one or both of those:

- **`categories`** — the fixed, staff-defined list `businesses` are filed under.
- **`projects`** — a portfolio/showcase item a business posts (title, description, image, link, completion date), each tied to a `business_id`. Goes through the same review queue as a new business/bot before it's public; posted by the business's verified owner or staff (see [routes/projects](routes/projects)).
- **`reviews`** — belongs to exactly one of a business, a bot or a project (DB `CHECK` constraint enforces it — counts how many of the three subject columns are set rather than a plain `<>`, now that there are three), with a 1–5 rating, title, body, optional owner reply, and a helpful/unhelpful vote tally. A project's replies come from its owning business's verified owner (a project has no `owner_id` of its own).
- **`review_votes`** — one row per user per review; later votes overwrite earlier ones.
- **`reports`** — filed against a review, business, or bot; staff resolve as resolved/dismissed.
- **`claims`** — an *ownership* claim ("this business is mine"), separate from a review-queue claim ("I'm reviewing this submission"). Staff approve/deny these to set a business's verified `owner_id`.
- **`moderation_actions`** — append-only history of every claim/unclaim/approve/deny, against a bot, a business, or a project, surfaced via `GET /actions`.
- **`users` / `discord_accounts`** — a Metro user is its own row, not a Discord account; `discord_accounts` links a Discord ID to one. See [identity/identity.go](identity/identity.go) — every account today is created via Discord OAuth, but the schema doesn't hard-code that.
- **`roles` / `user_roles`** — named, permission-bearing roles (see below), independent of the bot/business review system.
- **`webhooks`** — outbound event subscriptions (see below), registered against a `target_type`/`target_id` pair the same polymorphic way `reports`/`moderation_actions` are.

## The moderation queue (the core mechanic)

`bots`, `businesses` and `projects` all move through the same five-state pipeline, implemented once in [review/action.go](review/action.go) (`ApplyBotAction`/`ApplyBusinessAction`/`ApplyProjectAction`) and reused everywhere:

```
PENDING ──claim──▶ UNDER_REVIEW ──approve──▶ APPROVED ──(report resolved)──▶ SUSPENDED
                        │
                        └──deny──▶ DENIED
```

- Surfaced in Discord as `/queue` (paginated list; a select menu switches between the bot, business and project queues rather than merging them — see [bot/commands/queue.go](bot/commands/queue.go)) plus standalone `/claim /unclaim /approve /deny` commands.
- Surfaced over HTTP as `POST /bots/{id}/review/*`, `POST /businesses/{id}/review/*` and `POST /projects/{id}/review/*`.
- Every action is staff-gated (via [rpc.ReviewBot/ReviewBusiness/ReviewProject](rpc/review.go) on the HTTP side) and writes a `moderation_actions` row.
- Every `ApplyXAction` ends by calling a shared `finishAction` (commit, then `webhooks.DispatchQueueAction`) — so a future subject type gets both `moderation_actions` logging and webhook delivery for free just by ending its own apply function the same way.

**Reviews are not part of this pipeline** — this was a point you corrected me on for the profile README, worth restating here since it's the same fact: anyone can post a review the moment a business/bot is approved, no staff step required. Staff only get involved with a review *after the fact*, via reports.

## Roles & permissions

A newer system than the queue, and the thing most of the recent `?? ` (untracked) files are about ([roles/](roles/), [perms/](perms/), `routes/roles/`, `bot/commands/roles*.go`, `bot/commands/syncroles.go`):

- [perms/perms.go](perms/perms.go) defines a fixed catalog of 5 permissions (`panel.access`, `queue.review`, `reviews.moderate`, `claims.resolve`, `roles.manage`) plus a `*` wildcard.
- A **role** ([types.Role](types/types.go)) is a name + a permission set, optionally linked to a real Discord role ID.
- If a role is Discord-linked, its membership is kept in sync automatically — [roles.SyncMember](roles/roles.go) runs whenever a member's roles change in Discord and on every panel login; `/syncroles` (or `POST /roles/sync`) forces a full reconcile.
- A role with no linked Discord role is assigned by hand, in the panel or via `/roles` in Discord.
- `config.yaml`'s `owners` list is a permanent escape hatch — it bypasses this whole system (`state.Config.IsOwner`), so a fresh deploy with an empty `roles` table still has someone who can log in and set things up.

## Webhooks

Outbound event notifications, registered against any target — deliberately not limited to bots/businesses/projects: [webhooks.OwnsTarget](webhooks/ownership.go) and [webhooks.Dispatch](webhooks/dispatch.go) both key off a plain `target_type`/`target_id` string pair, so a future subject type needs no change here, just its own `ApplyXAction` calling `review.finishAction` (queue events) and/or its own routes calling `webhooks.Dispatch` (custom events).

- [webhooks/events.go](webhooks/events.go)'s `Catalog` is the fixed list of events a webhook can subscribe to: `review.created/updated/deleted/voted/responded` and `queue.claimed/unclaimed/approved/denied`. `GET /webhook-events` exposes it.
- A webhook's `Events` list is the subscription filter — empty means every event for that target.
- Registering one (`POST /webhooks`, target in the body) requires owning the target ([webhooks.OwnsTarget](webhooks/ownership.go): a business's `owner_id`, a project's owning business's `owner_id`, or a bot's `owner`/`extra_owners`) or a staff session — same rule `routes/reviews/response.go`'s owner-reply check uses (it now delegates to `OwnsTarget` instead of its own copy).
- Deliveries are an HTTP POST of `{id, event, target_type, target_id, created_at, data}`, HMAC-SHA256-signed over the raw body with the webhook's own secret (`X-Metro-Signature: sha256=<hex>`), fired in a goroutine — never blocks or fails the request that triggered it. `POST /webhooks/{id}/test` sends one synchronously instead, to verify setup.
- The secret is only ever shown once, in the `POST /webhooks` or `POST /webhooks/{id}/rotate-secret` response (`types.WebhookRevealed`) — every other response's `Webhook.Secret` is `json:"-"`.
- 10 consecutive delivery failures auto-disables a webhook (`enabled = false`) rather than retrying forever; there's no delivery log table yet, just `failure_count`/`last_triggered_at` on the row itself.

## Auth model — worth double-checking

There's no framework-level auth enforcement. [api/api.go](api/api.go)'s `authorize` callback (what the `uapi` router library calls before every request) unconditionally returns `Authorized: true`. The `Auth: []uapi.AuthType{{Type: "Staff"}}` annotations you see on routes are **documentation only** — actual enforcement is each handler manually calling one of [api/auth.go](api/auth.go)'s three helpers:

- `AuthUser` — valid session required.
- `AuthStaff` — `AuthUser` + `is_staff` (or config owner).
- `AuthPermission(ctx, r, perm)` — `AuthUser` + holds `perm` (or config owner).

I checked a sample of handlers and they match their declared `Auth` annotation, but because nothing enforces the two stay in sync, **it's worth a pass to confirm every route that declares `Staff`/`User` actually calls the matching helper** — a route that declares `Staff` but forgets the `AuthStaff()` call would silently be wide open.

---

## What's implemented (functionally complete, as far as the code shows)

- ✅ Businesses: submit, browse/search/sort, update, review queue, ownership claims
- ✅ Bots: submit, review queue (same pipeline as businesses)
- ✅ Projects: post (owner or staff), browse/sort per business, update, review queue (same pipeline as businesses/bots), reviews/ratings
- ✅ Categories: fixed list, browsable
- ✅ Webhooks: register/update/delete/rotate-secret/test against any target, signed delivery on review + queue events
- ✅ Reviews: post/edit/delete, helpful voting, owner responses, reporting — against a business, a bot, or a project
- ✅ Moderation action history (`/actions`)
- ✅ Roles & permissions: CRUD, assignment, Discord role sync (both directions)
- ✅ Discord bot: `/queue`, `/claim /unclaim /approve /deny`, `/roles`, `/syncroles`, `/support`, plus legacy `%`-prefix commands (`/invite` and `/sync` were removed)
- ✅ Panel login: Discord OAuth2 → nonce ticket → session token
- ✅ Team endpoint (`GET /team` — who currently holds reviewer permissions)
- ✅ DB migrations 0001–0007, applied automatically on startup, with the old bot-only schema (`bot_list`/`bot_queue`/`bot_action`, the `silverpelt` fanout) fully retired
- ✅ `cmd/seed` — idempotent local dev data seeding, including a `-reset` flag

## What's genuinely missing or unverified

- ❌ **No automated tests** outside the `migrations` package. Nothing covers the review queue state machine, roles/permissions, or any HTTP handler.
- ❌ **No developer-facing README** in this repo (the `PROFILE_README.md` we wrote is for a GitHub profile page, not this repo — different audience, deliberately code-free).
- ❌ **No CI** (`.github/workflows` doesn't exist).
- ⚠️ **Config has nothing filled in** — [config.yaml.sample](config.yaml.sample) has every Discord/auth field blank; a real deploy needs a filled `config.yaml` (gitignored) before anything runs.
- ⚠️ **Auth annotations vs. enforcement** — see above, worth an audit pass.
- ⚠️ **This is early history** — only 4 commits total on `main`, so there's not much git archaeology to lean on if something looks unfinished; what you see in the working tree is closer to "the whole story" than usual.
- ❓ **Where does the panel/frontend live?** Not in this repo. If it's not started yet, that's the other half of "usable product" this backend is waiting on.

---

## Suggested next steps, roughly in order

1. **Fill in `config.yaml`** from the sample and confirm the service boots against a real (or local) Postgres + Discord bot token — that's the fastest way to confirm the above "implemented" list actually works end to end.
2. **Decide on a test baseline** — at minimum, the `review` package's state machine and `roles`/`perms` permission checks are cheap to unit test and are the highest-consequence logic in the repo.
3. **Audit auth annotations vs. handlers** — a quick pass confirming every `Auth: []uapi.AuthType{...}` route actually calls the matching `api.AuthUser`/`AuthStaff`/`AuthPermission`.
4. **Write the real repo README** (developer-facing: setup, env vars, `go run`, migrations, seeding) — separate from the profile one.
5. **Figure out the panel situation** — confirm whether it's a separate repo already underway, or still to be started.

## Since the last update

- **Licensed MIT.** [LICENSE](LICENSE) now exists (Purrquinox Digital, 2026), and [api/api.go](api/api.go)'s advertised license was corrected from a nonexistent `AGPL-3.0` reference to match.
- **Restructured for maintainability + speed.** Every file that had grown past ~250–300 lines and mixed concerns got split along its natural seams (`types/types.go` → 7 domain files, `roles/roles.go` → 5 concern files, `cmd/seed/main.go` → 6 files, several `bot/commands/*.go` and all four `routes/*` route-wiring files). A new [migrations/0008_performance_indexes.sql](migrations/0008_performance_indexes.sql) adds composite/trigram indexes matching the exact query shapes `getAllBusinesses`/review-fetch handlers issue, and `state.Setup()` now tunes the Postgres pool explicitly (`database.max_conns`/`min_conns` in config) instead of relying on library defaults. No behavior changes — verified with `go build`/`go vet`/`gofmt`/`go test` after every step.
