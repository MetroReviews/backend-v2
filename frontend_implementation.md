# Metro Reviews — Frontend Implementation Guide

This document describes how a frontend should be built (or rebuilt) against the Metro Reviews
backend (`backend-v2`). It is written to be a **complete, self-sufficient reference**: an LLM or
engineer with no access to the Go source should be able to read this file and implement a
frontend that uses every feature the API exposes, correctly.

Metro Reviews is a Yelp-style review platform: businesses are listed under categories, businesses
can post "projects" (case studies / portfolio items / menu items — a sub-entity that can also be
reviewed independently), users leave star ratings + text reviews, business owners can claim and
respond to reviews, and a staff team moderates everything through an approval queue. There is also
a Discord bot/panel side (out of scope for the public frontend, described briefly for context).

There is currently no frontend in this repository — this is the spec to build one from scratch.
I'
---

## 1. High-level architecture

```
Browser (your frontend) ──HTTP/JSON──▶ Go API (chi router, this repo) ──▶ Postgres
                                              │
                                              ├─▶ Redis (optional: cache + rate limiting)
                                              ├─▶ Discord API (OAuth login, role sync, bot)
                                              └─▶ SMTP (optional: review-invite emails)
```

- **Transport**: plain REST over JSON. No GraphQL, no websockets. `Content-Type: application/json` on every request/response.
- **Base URL**: `http://<server.bind_addr>` in dev (see `config.yaml`, default `:8080`), or `https://<auth.allowed_host>` in prod. There is no `/api` or `/v1` path prefix — routes are mounted directly at root (e.g. `/businesses`, not `/api/v1/businesses`).
- **CORS**: wide open. The server echoes back whatever `Origin` header it receives and sets `Access-Control-Allow-Credentials: true`, so the frontend can be hosted anywhere without a proxy.
- **OpenAPI**: a live schema is served at `GET /openapi` and human docs at `GET /docs` (Redoc). Regenerate frontend API-client types from `/openapi` if you have a codegen step — the shapes documented below are guaranteed to match it exactly, since it's generated from the same struct tags.
- **IDs**: everything is a UUID (v4) except Discord IDs (int64, serialized as strings in JSON) and role→permission slugs (plain strings).

---

## 2. Authentication

There is no cookie session — the frontend owns the session token and sends it explicitly.

### 2.1 Session token

All authenticated requests send:

```
Authorization: Bearer <session_token>
```

Tokens are opaque random strings, **valid for 30 days** from issue, and are returned by every
login/register endpoint alongside an `expires_at` timestamp. Store the token (e.g.
`localStorage` or a secure cookie you manage yourself) and the `expires_at`; treat the token as
dead once expired and force a re-login (the API will also just start returning 401s).

There is no refresh-token flow — when a token expires, the user logs in again. Design the
frontend to react to a 401 on any authenticated call by clearing local session state and routing
to login, rather than trying to pre-empt expiry.

### 2.2 Three ways to authenticate

**A. Discord OAuth (first-party client running its own OAuth flow)**

The frontend runs a normal Discord OAuth2 `identify` (and ideally `guilds`) authorization-code
flow itself, obtains a Discord access token, then exchanges it:

```
POST /auth/login
{ "access_token": "<discord access token>", "token_type": "Bearer" }
→ 200 { "session_token", "expires_at", "user_id" (Discord ID), "username", "avatar" }
```

This creates the Metro user on first login (`identity.EnsureDiscordUser`), updates
username/avatar, and syncs Discord role → Metro role assignments if a bot is configured. Rate
limited to 20/hour per client.

**B. Email + password — register**

```
POST /auth/register
{ "email", "password" (8-72 chars), "username"?: optional display name }
→ 200 { "session_token", "expires_at", "user_id", "email", "username" }
→ 409 if the email is already registered
```
Rate limited to 5/hour.

**C. Email + password — login**

```
POST /auth/login/password
{ "email", "password" }
→ 200 { "session_token", "expires_at", "user_id", "email", "username" }
→ 401 "Invalid email or password" (deliberately identical whether the email exists or the
   password is wrong — timing-safe, don't try to distinguish these cases in the UI)
```
Rate limited to 10/15min.

**D. Attach a password to an already-logged-in account**

Lets a user who logged in via Discord also set an email/password, so either method works
afterward. Requires an existing session:

```
POST /auth/password          (Authorization: Bearer <token> required)
{ "email", "password" (8-72 chars) }
→ 200 { "message": "Password set", "error": false }
→ 409 if that email is already registered to a *different* account
```

There is no separate "logout" endpoint — logging out client-side is just discarding the stored
token (the session row still exists server-side until it expires; that's fine).

### 2.3 What a session buys you

Three auth tiers are used across the API (see `Auth:` on each route below):

| Tier | Meaning |
|---|---|
| *(none)* | Public, no `Authorization` header needed |
| **User** | Any valid, non-banned session — `Authorization: Bearer <token>` |
| **Staff** | A `User` session where `is_staff = true`, OR the caller's linked Discord ID is in the configured owner list |
| **Permission-gated** | A subset of Staff-tier routes additionally require one specific permission slug (e.g. `roles.manage`) — see §7 |

A banned user (`banned: true` in the `User` shape, §3) gets a 403 `"This account is banned"` on
**every** authenticated call, including `GET /me` itself — even though their token is otherwise
valid. There's no successful response to read `banned` off; treat that specific 403 message as the
signal and surface it as an "account suspended" state, not a generic error.

### 2.4 Fetching "who am I"

Two calls make up your "who am I" bootstrap on app load:

```
GET /me                    (User)
→ User                     // username, avatar, bio, is_staff, banned, discord_id — see §3

GET /me/permissions        (User)
→ { "roles": Role[], "permissions": string[] }
```

Call both on app load (or on a token you loaded from storage) to hydrate the current user's
profile and permissions in one round trip each; if either 401s, the stored token is
invalid/expired — clear it and route to login. `permissions` is the flattened union of every role
the user holds; `"*"` in that array means the user can do literally anything permission-gated
(super-admin / config-owner equivalent). Cache both client-side after login and after any profile
or role change; there's no push mechanism, so re-fetch on app load and after admin actions.

Note `GET /me` shares the same auth check as every other authenticated route (`api.AuthUser`), so
a banned user gets the same 403 `"This account is banned"` from it as from anything else — it's
not a safe way to detect banned-ness up front, just the normal profile fetch. You don't strictly
need to capture username/avatar off the login response anymore (§2.2) — `GET /me` is the source of
truth for the current user's profile — but the login response still returns them inline too, so a
first paint doesn't have to wait on a second request.

---

## 3. Core data model

TypeScript-shaped reference for every entity the API returns. `db`/`json` tags in the Go source
are identical field names — use these 1:1 for API client types.

```ts
type UUID = string;
type ISODateTime = string;

// Enums are transmitted as small integers, not strings. Map them client-side.
enum State { Pending = 0, UnderReview = 1, Approved = 2, Denied = 3, Suspended = 4 }
enum ReviewStatus { Published = 0, Flagged = 1, Removed = 2 }
enum ReportStatus { Open = 0, Resolved = 1, Dismissed = 2 }
enum ClaimStatus { Pending = 0, Approved = 1, Denied = 2 }
enum ModerationActionType { Claim = 0, Unclaim = 1, Approve = 2, Deny = 3 }
enum ReviewInviteStatus { Pending = 0, Redeemed = 1, Expired = 2 }

interface Category {
  id: UUID; slug: string; name: string;
  description: string | null; icon: string | null;
}

interface Business {
  id: UUID; category_id: UUID; slug: string; name: string;
  description: string | null; website: string | null; logo: string | null; banner: string | null;
  address: string | null; city: string | null; country: string | null;
  metadata: Record<string, any>;             // free-form, category-specific fields
  owner_id: UUID | null;                     // set only once an ownership claim is approved
  submitted_by: UUID;
  status: State;
  reviewer: UUID | null;                     // staff member who claimed it for review
  avg_rating: number; review_count: number;
  created_at: ISODateTime; updated_at: ISODateTime;
  latitude: number | null; longitude: number | null;
  gallery: string[];                         // showcase photos, separate from logo/banner
  featured: boolean; featured_until: ISODateTime | null;
  view_count: number;
}

// Returned only from GET /businesses (the list/search endpoint) — same as Business plus:
interface BusinessSearchResult extends Business {
  distance_km?: number | null;               // present only when lat/lng were passed
}

interface Project {
  id: UUID; business_id: UUID; title: string; description: string | null;
  image: string | null; url: string | null; completed_at: ISODateTime | null;
  submitted_by: UUID; reviewer: UUID | null; status: State;
  avg_rating: number; review_count: number;
  created_at: ISODateTime; updated_at: ISODateTime;
}

interface Review {
  id: UUID;
  business_id: UUID | null; project_id: UUID | null;   // exactly one is set
  author_id: UUID;
  rating: number;                                        // 1-5
  title: string | null; body: string;
  owner_response: string | null; owner_response_at: ISODateTime | null;
  helpful_count: number;                                 // net helpful votes (helpful - unhelpful)
  status: ReviewStatus;
  created_at: ISODateTime; updated_at: ISODateTime;
  photos: string[];
  flag_reason?: string;                                   // present only if fraud-flagged; omitted otherwise
  verified: boolean;                                      // true if posted via a redeemed invite token
}

interface ReviewInvite {
  id: UUID; business_id: UUID | null; project_id: UUID | null;
  target_email: string; token: string;                    // pass `token` back as `invite_token` on POST /reviews
  created_by: UUID; status: ReviewInviteStatus;
  redeemed_review_id: UUID | null;
  expires_at: ISODateTime; created_at: ISODateTime;
}

interface Claim {                                          // ownership claim on a business
  id: UUID; business_id: UUID; user_id: UUID; note: string | null;
  status: ClaimStatus; created_at: ISODateTime;
  resolved_by: UUID | null; resolved_at: ISODateTime | null;
}

interface Report {                                          // "flag this content" report
  id: UUID; target_type: "review" | "business" | "project"; target_id: string;
  reporter_id: UUID; reason: string; status: ReportStatus;
  created_at: ISODateTime; resolved_by: UUID | null; resolved_at: ISODateTime | null;
}

interface ModerationAction {
  id: UUID; target_type: "business" | "project"; target_id: string;
  action: ModerationActionType; reason: string; reviewer: UUID; action_time: ISODateTime;
}

interface Role {
  id: UUID; name: string; discord_role_id: string | null;   // linked Discord role, if synced
  permissions: string[];                                     // "*" = every permission
  created_at: ISODateTime; updated_at: ISODateTime;
}

interface Permission { slug: string; description: string; }

interface TeamMember {
  username: string; id: string /* Discord ID */; avatar: string;
  is_list_owner: boolean; sudo: boolean; roles: string[];
}

interface Webhook {
  id: UUID; target_type: string; target_id: string; url: string;
  events: string[];               // empty = subscribed to everything
  created_by: UUID; enabled: boolean; failure_count: number;
  last_triggered_at: ISODateTime | null;
  created_at: ISODateTime; updated_at: ISODateTime;
  // NOTE: `secret` is never included here — see §9.
}
interface WebhookRevealed extends Webhook { secret: string; }   // only on create + rotate-secret
interface WebhookEvent { name: string; description: string; }

interface SearchResult {
  type: "business" | "project"; id: UUID;
  slug: string | null;                 // businesses only
  name: string;                        // business name, or project title
  description: string | null;
  avg_rating: number; review_count: number;
  rank: number;                        // full-text rank, higher = more relevant; sort key, don't display
}

interface User {                       // GET /me returns this for the caller — see §2.4
  id: UUID; discord_id: string | null; username: string | null; avatar: string | null;
  bio: string | null; is_staff: boolean; banned: boolean; created_at: ISODateTime;
}

// Generic envelopes used across many endpoints:
interface ApiError { message: string; error: boolean; context?: Record<string, string>; }
interface UpdatedResponse { has_updated: string[]; }   // PATCH endpoints return which fields changed
```

**Error responses are always `ApiError`-shaped**, on every non-2xx status, with `error: true`. A
generic API client should treat any response whose JSON has `"error": true` — or any non-2xx
status — as a failure and surface `message` to the user. `context` is populated for per-field
validation failures.

**Enum values are integers, not strings.** Build a shared enum→label mapping in the frontend
(e.g. `STATE_LABELS = { 0: "Pending", 1: "Under Review", 2: "Approved", 3: "Denied", 4: "Suspended" }`)
rather than hardcoding numbers in components.

---

## 4. Endpoint reference

All paths are relative to the base URL. "Auth" column: blank = public, **User** = any logged-in
user, **Staff** = staff/owner, **Staff\*** = staff tier *and* a specific permission (named).

### 4.1 Auth

| Method | Path | Auth | Notes |
|---|---|---|---|
| POST | `/auth/login` | | Discord access-token exchange → session. §2.2A |
| POST | `/auth/register` | | Email/password signup → session. §2.2B |
| POST | `/auth/login/password` | | Email/password login → session. §2.2C |
| POST | `/auth/password` | User | Attach password to current account. §2.2D |
| GET | `/me` | User | Caller's own profile: `username`, `avatar`, `bio`, `is_staff`, `banned`, `discord_id`. Your app's "who am I" bootstrap call — see §2.4. |

### 4.2 Categories

| Method | Path | Auth | Notes |
|---|---|---|---|
| GET | `/categories` | | Every business category, alphabetical. Small, static-ish list — fetch once and cache (e.g. on app boot) for building category filter UI, category-picker dropdowns on the "add business" form, etc. |

### 4.3 Businesses

| Method | Path | Auth | Notes |
|---|---|---|---|
| GET | `/businesses` | | Browse/search/filter (see query params below). Returns `BusinessSearchResult[]`. |
| GET | `/businesses/{slug}` | | Single business by URL slug. 404s if not approved *and* caller isn't staff. Records a page view (fire-and-forget) when approved. |
| POST | `/businesses` | User | Submit a new business. Enters the moderation queue at `Pending`. |
| PATCH | `/businesses/{id}` | User\* | Update fields. \*Caller must be the verified `owner_id` or staff. Returns `UpdatedResponse`. |
| GET | `/businesses/{id}/similar` | | Up to 6 approved businesses, same category, best-rated first. |
| GET | `/me/recommended` | User | Businesses recommended from the caller's review history's categories (falls back to overall top-rated for new users). |
| GET | `/me/businesses` | User | Every business the caller owns (`owner_id`) or originally submitted (`submitted_by`), **any moderation status** — this is the listing call an owner dashboard's "My Businesses" page is built around. `offset`/`limit` (default 20, max 100). |
| GET | `/businesses/{id}/analytics` | User\* | Owner/staff only. View count, 12-week review trend, rating distribution — see §4.3.1. |
| GET | `/businesses/{id}/widget` | | Public, cacheable embed payload — see §10. |
| PATCH | `/businesses/{id}/feature` | Staff\* (`businesses.feature`) | Toggle sponsored/featured placement. Data-model only, no payment. |
| POST | `/businesses/{id}/invites` | User\* | Owner/staff creates a review-invite email. See §8.3. |
| GET | `/invites/{token}` | | Resolve an invite token (public — the token *is* the credential). |
| POST | `/businesses/{id}/review/claim` | Staff | Moderation queue: claim for review. Body: `{ "reason": string (min 5 chars) }`. |
| POST | `/businesses/{id}/review/unclaim` | Staff | Release back to pending. |
| POST | `/businesses/{id}/review/approve` | Staff | Publish it. |
| POST | `/businesses/{id}/review/deny` | Staff | Deny it. |
| POST | `/businesses/{id}/claim` | User | File an *ownership* claim (different from the moderation "claim for review" above!). Body: `{ "note"?: string }`. |
| POST | `/businesses/{id}/claims/{claim_id}/approve` | Staff\* (`claims.resolve`) | Approves ownership → sets `owner_id`. |
| POST | `/businesses/{id}/claims/{claim_id}/deny` | Staff\* (`claims.resolve`) | Denies ownership claim. |
| GET | `/me/claims` | User | Every ownership claim the caller has filed, any status, newest first — so a dashboard can show "pending"/"approved"/"denied" without polling individual businesses. `offset`/`limit` (default 20, max 100). |

⚠️ **Naming collision to be careful about in the UI/routing layer**: `POST /businesses/{id}/claim`
(singular, ownership claim by any user) and `POST /businesses/{id}/review/claim` (staff claiming
the moderation queue item) are completely different actions. Name your frontend functions/routes
distinctly (e.g. `claimOwnership()` vs `claimForModerationQueue()`) to avoid mixing them up.

**`GET /businesses` query parameters** (all optional except none are required):

| Param | Type | Effect |
|---|---|---|
| `category` | string | Filter by category slug |
| `q` | string | Name/description ILIKE + full-text match |
| `city` | string | ILIKE match |
| `country` | string | ILIKE match |
| `min_rating` | float | `avg_rating >= min_rating` |
| `lat`, `lng` | float | If both given, adds a `distance_km` field to every result (Haversine, km) |
| `radius_km` | float | Only usable together with `lat`/`lng` — excludes anything farther, and excludes businesses with no coordinates |
| `sort` | `new` (default) \| `rating` \| `reviews` \| `distance` | `distance` only makes sense with `lat`/`lng` set |
| `all` | `"true"` | Staff only (silently ignored otherwise) — includes non-approved businesses |
| `offset`, `limit` | int | Pagination, default 20 / max 100 |

Featured businesses (`featured: true`) always sort first regardless of `sort`, then the secondary
sort applies — reflect this in listing UI (e.g. a "Sponsored" badge/section at the top).

#### 4.3.1 Business analytics response

```ts
GET /businesses/{id}/analytics →
{
  view_count: number;
  review_trend: { week_start: ISODateTime; count: number }[];    // last 12 weeks
  rating_distribution: { rating: number; count: number }[];      // one bucket per star 1-5
}
```
Good fit for an owner dashboard: a line/bar chart of `review_trend`, a bar chart of
`rating_distribution`, and a KPI tile for `view_count`.

### 4.4 Projects

Projects are sub-entities of a business (e.g. a specific product, menu item, or case study) that
carry their own rating and go through the *same* moderation queue shape as businesses.

| Method | Path | Auth | Notes |
|---|---|---|---|
| GET | `/businesses/{business_id}/projects` | | List a business's projects. `sort=new\|rating\|reviews`, `all=true` (staff). |
| GET | `/projects/{id}` | | Single project. |
| POST | `/businesses/{business_id}/projects` | User\* | Owner/staff only. Enters queue at `Pending`. |
| PATCH | `/projects/{id}` | User\* | Owner (of the parent business)/staff only. Returns `UpdatedResponse`. |
| POST | `/projects/{id}/review/claim` | Staff | Same claim/unclaim/approve/deny queue shape as businesses. |
| POST | `/projects/{id}/review/unclaim` | Staff | |
| POST | `/projects/{id}/review/approve` | Staff | |
| POST | `/projects/{id}/review/deny` | Staff | |

There is no `all=true`-style unapproved single-project fetch guard documented separately — like
businesses, treat a non-staff fetch of a non-approved project's detail page as "not found" (the
list endpoint filters it out via `status`; build the project detail page to handle a 404-shaped
"pending" state gracefully, e.g. "This listing is awaiting approval").

### 4.5 Reviews

| Method | Path | Auth | Notes |
|---|---|---|---|
| POST | `/reviews` | User | Create. Body includes exactly one of `business_id`/`project_id`. One review per user per subject — a second attempt errors. |
| GET | `/businesses/{id}/reviews` | | Published reviews, newest first. `offset`/`limit` (default 20, max 100). |
| GET | `/projects/{id}/reviews` | | Same, for a project. |
| GET | `/me/reviews` | User | Every review the caller has authored, across all businesses/projects, **in any status** (including `Flagged`/`Removed`, unlike the two public list endpoints above which only return `Published`). Newest first. `offset`/`limit` (default 20, max 100). |
| PATCH | `/reviews/{id}` | User\* | Author only. Partial update (`rating`/`title`/`body`). |
| DELETE | `/reviews/{id}` | User\* | Author or staff. |
| POST | `/reviews/{id}/vote` | User | `{ "helpful": boolean }`. One vote per user per review — a later vote overwrites the earlier one (don't build an "already voted, can't change it" restriction into the UI; instead reflect the latest choice as a toggle). |
| POST | `/reviews/{id}/response` | User\* | Owner (of the reviewed business/project) or staff posts/replaces the official reply. `{ "response": string }`. |
| POST | `/reviews/{id}/report` | User | Flag a review for staff. `{ "reason": string }` → returns a `Report`. |

**`ReviewCreate` body:**
```ts
{
  business_id?: UUID; project_id?: UUID;   // exactly one required
  rating: number;                          // 1-5, required
  title?: string;
  body: string;                            // required
  photos?: string[];                       // https URLs, max 6
  invite_token?: string;                   // redeem a review-invite (see §8.3) → marks review "verified"
}
```

A submitted review can silently come back with `status: Flagged` and a `flag_reason` populated
instead of `Published` — the fraud package (§8.4) auto-flags likely spam/duplicate content. If
you show the caller their own just-submitted review, check `status`/`flag_reason` and, if flagged,
tell them it's pending a manual look rather than showing it as live (flagged reviews don't count
toward `avg_rating`/`review_count`, and don't show up in the public `GET .../reviews` list — only
`status = Published` reviews do).

### 4.6 Search

| Method | Path | Auth | Notes |
|---|---|---|---|
| GET | `/search?q=...` | | Ranked full-text search across **approved businesses and projects together**. `offset`/`limit` (default 20, max 100). Results are cached server-side for 30s per exact query string. |

This is the single global search box endpoint — distinct from `GET /businesses?q=...`, which only
searches businesses and supports the full filter set (category/city/rating/location). Use
`/search` for a top-nav "search everything" box that mixes businesses and projects into one
ranked feed (branch UI per-row on `type`); use `/businesses?q=...&category=...&...` for a
dedicated business-directory/browse page with filters.

### 4.7 Team

| Method | Path | Auth | Notes |
|---|---|---|---|
| GET | `/team` | | Public "our review team" page data: every user holding `queue.review`, with Discord avatar, list-related role names, and `is_list_owner`/`sudo` flags. |

### 4.8 Moderation action history

| Method | Path | Auth | Notes |
|---|---|---|---|
| GET | `/actions` | | Public audit log of claim/unclaim/approve/deny actions. `target_type=business\|project`, `offset`/`limit` (default 50, max 200). Good for a public "moderation transparency" log page. |

### 4.9 Roles & permissions (staff panel)

| Method | Path | Auth | Notes |
|---|---|---|---|
| GET | `/permissions` | Staff | The fixed permission catalog (see §7 table) — for building a role-editor's checkbox list. |
| GET | `/me/permissions` | User | Caller's own roles + effective permissions. Pair with `GET /me` (§4.1) for profile fields — together they're your app's auth/bootstrap call (§2.4). |
| GET | `/roles` | Staff | All roles. |
| POST | `/roles` | Staff\* (`roles.manage`) | `{ name, discord_role_id?, permissions[] }` |
| PATCH | `/roles/{id}` | Staff\* (`roles.manage`) | Partial update, same shape, all fields optional. Passing `discord_role_id: ""` unlinks Discord sync. |
| DELETE | `/roles/{id}` | Staff\* (`roles.manage`) | Deletes the role and every assignment to it. |
| PUT | `/roles/{id}/members/{user_id}` | Staff\* (`roles.manage`) | Grant a role to a (Metro) user ID. ⚠️ If the role is Discord-linked, the next Discord role sync will overwrite this unless the user also holds the linked Discord role — warn admins of this in the UI when they manually assign a Discord-linked role. |
| DELETE | `/roles/{id}/members/{user_id}` | Staff\* (`roles.manage`) | Revoke. |
| POST | `/roles/sync` | Staff\* (`roles.manage`) | Force a full resync of Discord-linked role membership from the guild (normally automatic on bot events). |

### 4.10 Webhooks

See §9 for the full integration story (signing, events, target ownership).

| Method | Path | Auth | Notes |
|---|---|---|---|
| GET | `/webhook-events` | User | Fixed catalog of subscribable event names. |
| GET | `/webhooks?target_type=...&target_id=...` | User\* | List webhooks on one target. Owner/staff only. |
| POST | `/webhooks` | User\* | Register. Returns `WebhookRevealed` — **the only time `secret` is ever sent**. |
| PATCH | `/webhooks/{id}` | User\* | Update `url`/`events`/`enabled`. |
| DELETE | `/webhooks/{id}` | User\* | Unregister. |
| POST | `/webhooks/{id}/rotate-secret` | User\* | New secret, old one invalidated immediately. Also only shown once. |
| POST | `/webhooks/{id}/test` | User\* | Fire an immediate signed test delivery bypassing the event filter. |

### 4.11 Panel (Discord-bot-adjacent, internal)

These back the separate staff Discord-OAuth panel login flow, not the main app:

| Method | Path | Auth | Notes |
|---|---|---|---|
| GET | `/_panel/strikestone` | | Returns the Discord OAuth2 authorize URL to redirect a staff member to. |
| GET | `/_panel/frostpaw?code&state` | | OAuth2 callback — redirects to `{state}/login?ticket=...` (a base64 ticket, not a bearer token, containing a one-time nonce + an embedded `session_token`). |
| GET | `/_panel/mapleshade?ticket=...` | | Verifies a ticket + guild-membership + staff-permission, one-time. |

Treat these as belonging to a *separate, internal* panel app, not the public reviews frontend —
mentioned here for completeness only. If you *are* building that internal panel, its login flow
is: redirect to `/_panel/strikestone`'s URL → Discord → back to your panel's `/login?ticket=...` →
call `/_panel/mapleshade?ticket=...` → on `access: true`, decode the ticket's embedded
`session_token` (it was base64'd into the redirect ticket) and use it as the normal
`Authorization: Bearer` session token for every other endpoint in this document.

---

## 5. The moderation queue (businesses & projects)

Both businesses and projects share one state machine:

```
Pending ──claim──▶ UnderReview ──approve──▶ Approved
                        │
                        ├──deny────────────▶ Denied
                        └──unclaim─────────▶ Pending (back to start)

(Suspended exists as a status value but nothing in this API transitions into it automatically —
 build it as a manual/reserved state if your product needs it; no endpoint currently sets it.)
```

- A brand-new business/project starts at `Pending` and is invisible to the public (`GET` list/detail
  endpoints filter to `status = Approved` for non-staff callers).
- Only a staff session can drive the queue (`/{...}/review/claim|unclaim|approve|deny`), and each
  transition requires a `{ "reason": string }` body of **at least 5 characters** — build the staff
  UI so the action button is disabled until a reason of sufficient length is typed.
  Invalid transitions (e.g. approving something still `Pending`) 4xx with a message like *"This
  cannot be approved as it is not under review."* — surface that message directly, it's already
  user-facing.
  - `claim`: `Pending → UnderReview`, records `reviewer` as the caller
  - `unclaim`: `UnderReview → Pending`
  - `approve`: `UnderReview → Approved` (now publicly visible)
  - `deny`: `UnderReview → Denied`
- Every transition is recorded in the public `GET /actions` audit log and fires a `queue.*`
  webhook event.
- A **staff queue UI** should be built around: `GET /businesses?all=true&sort=new` (and the
  projects equivalent) to list everything including pending/under-review items, with per-item
  claim/approve/deny buttons gated on the caller being staff (check `GET /me/permissions` →
  presence of `queue.review` or `"*"`).

Ownership claims (`POST /businesses/{id}/claim` + `.../claims/{claim_id}/approve|deny`) are a
**separate, unrelated workflow** — a regular user asserting "I own this already-approved
business," reviewed by staff holding `claims.resolve`. Don't conflate this claim queue with the
moderation queue above in your routing/UI.

---

## 6. Ownership model

`Business.owner_id` (nullable) is the source of truth for "who manages this listing." It's set
only when staff approves a `Claim` (§4.3, §5). Until then, `submitted_by` recorded who originally
created the listing, but that person does **not** get owner-level edit rights automatically —
only an approved claim (or being staff) grants access to:

- `PATCH /businesses/{id}` / `PATCH /projects/{id}`
- `GET /businesses/{id}/analytics`
- `POST /businesses/{id}/invites`
- `POST /reviews/{id}/response` (for reviews on that business/its projects)
- `PATCH /businesses/{id}/feature` (staff + `businesses.feature` permission only, not owner)
- Webhook management scoped to that `target_type`/`target_id`

Frontend implication: gate "Edit business," "Reply to review," "View analytics," "Invite
reviewers," and "Manage webhooks" UI on `business.owner_id === currentUser.id || currentUser.is_staff`
— don't rely on the API to hide the buttons, but do rely on it to reject the request
(403 `"You do not own this business"`) if the client-side check is ever wrong or stale.

To find *which* businesses to even show a dashboard for in the first place — the nav-level "does
this user have a dashboard at all, and for what" question — call `GET /me/businesses` (§4.3), not
a per-business owner check. It returns every business the caller owns or submitted regardless of
status, so it also doubles as the "track my pending submission" view for someone who hasn't been
approved as owner yet.

---

## 7. Permissions catalog

Fixed, closed set (`GET /permissions`, staff-only, for building a role editor):

| Slug | Grants |
|---|---|
| `panel.access` | Log into the staff panel |
| `queue.review` | Claim, unclaim, approve, deny businesses and projects |
| `reviews.moderate` | Flag/remove reviews and resolve reports |
| `claims.resolve` | Approve or deny business ownership claims |
| `roles.manage` | Manage roles and permissions, including Discord role sync |
| `businesses.feature` | Toggle a business's sponsored/featured placement |

`"*"` (wildcard) on a role grants all of the above and satisfies every `Staff\*` check. A user is
also implicitly full-staff (bypasses all permission checks) if their linked Discord ID is in the
server's configured owner list — there's no API-visible flag for this beyond it just working; don't
try to special-case it in the frontend, just trust `GET /me/permissions` and 403 responses.

Build permission-gated UI as: fetch `GET /me/permissions` once per session → check
`permissions.includes("*") || permissions.includes("<needed-slug>")` before rendering
staff-only controls (create/edit role, feature toggle, resolve claim, moderate review, etc).

---

## 8. Reviews feature depth

### 8.1 Voting
`POST /reviews/{id}/vote { helpful: boolean }` — idempotent-per-user upsert; calling it again with
a different value flips the caller's vote. `Review.helpful_count` is the *net* score
(helpful − unhelpful), not a raw count — display it as "N people found this helpful" only if you
also track/display totals yourself; the API doesn't expose separate helpful/unhelpful totals, only
the net.

### 8.2 Owner responses
`POST /reviews/{id}/response { response: string }` — owner/staff only; **replaces** any existing
response (it's an upsert, not an append/thread). Model this as a single editable "Owner's reply"
block under each review, not a comment thread.

### 8.3 Review invitations (verified reviews)
Flow for a business owner soliciting a review from a real customer, and marking it "verified":

1. Owner/staff: `POST /businesses/{id}/invites { target_email }` → creates a `ReviewInvite`,
   best-effort emails the link (no-op if SMTP isn't configured server-side — don't assume the
   email always sends; show the returned `token`/build-your-own link as a fallback "copy link"
   action in the UI regardless).
2. Recipient visits a frontend route like `/invite/{token}`, which calls
   `GET /invites/{token}` to resolve it (shows which business/project, checks `status`/`expires_at`
   client-side to show "expired" state).
3. Recipient submits `POST /reviews` with `invite_token` set to that token. On success the review
   comes back `verified: true` and the invite flips to `Redeemed`.

Build a "Verified Reviewer" badge on any `Review` with `verified: true`.

### 8.4 Fraud/spam/content-moderation signals (read-only from the frontend's perspective)
The backend auto-flags reviews server-side on both create (`POST /reviews`) and edit
(`PATCH /reviews/{id}`, when `title`/`body` is part of the edit); the frontend doesn't call
anything here directly, but should account for the behavior. On create, checks run in this order —
the first one that trips wins and short-circuits the rest, so `flag_reason` always names exactly
one cause:

1. **OpenAI Moderation API** (`moderation` package) — the review's title+body is sent to
   `POST https://api.openai.com/v1/moderations` (model `omni-moderation-latest`). If OpenAI flags
   it (hate, harassment, sexual, violence, self-harm, etc. categories), `flag_reason` comes back as
   `"flagged by content moderation: <category, category, ...>"`.
2. Near-duplicate text from the same author within 30 days (Postgres trigram similarity) →
   `flag_reason: "near-duplicate of another recent review by the same author"`.
3. Accounts younger than 10 minutes → `flag_reason: "posted by a newly created account"`.
4. **Semantic duplicate detection** (`fraud.IsSemanticDuplicate`) — the review body is embedded
   (`POST https://api.openai.com/v1/embeddings`, model `text-embedding-3-small`) and compared by
   cosine similarity against every other review on the *same business/project* from the last 30
   days, regardless of author. Catches a spam campaign spreading one reworded review across many
   different accounts, which the same-author-only check in step 2 structurally can't see.
   `flag_reason: "near-duplicate of another recent review on this listing (semantic match)"`.
5. **LLM authenticity classifier** (`fraud.ClassifyAuthenticity`) — a chat-completion call (model
   `gpt-4.1-mini`, structured JSON output) reads the rating + title + body and judges whether it
   reads as templated/incentivized/bot-written rather than genuine (no specific details, overtly
   promotional language, a reciprocal-review ask, or a rating/sentiment mismatch).
   `flag_reason: "flagged by AI authenticity check: <one-sentence reason from the model>"`. This
   is the priciest and slowest check (a full chat completion, run only after every cheaper check
   above has passed) — don't assume it runs synchronously-instant; it's still part of the same
   `POST /reviews` request/response cycle, it's just the tail latency contributor.

Steps 1, 4 and 5 are **entirely optional server-side config** — all three are gated on a single
`openai.api_key` in `config.yaml`. If it's unset, or any individual OpenAI call errors/times out,
that check fails open silently (never blocks or delays the review) and evaluation just falls
through to the next check in the list. Don't build the frontend to assume any OpenAI-backed check
always runs, and don't assume `flag_reason` is drawn from a small fixed enum — categories 1 and 5
are free-form text sourced from a live model call, not a hardcoded string, so render `flag_reason`
as opaque human-readable text (e.g. in a staff queue) rather than pattern-matching on it.

A flagged review has `status: Flagged` + a human-readable `flag_reason` and is excluded from
public listing and rating averages until staff resolves it via the reports/moderation tooling. If
you show the caller their own just-flagged review (see §4.5's `ReviewCreate` note above), the
`flag_reason` string is safe to show verbatim as an explanation.

**Editing an already-published review re-runs a subset of these checks on the new text**: the
OpenAI moderation check (step 1) and the semantic-duplicate check (step 4), but *not* the
same-author trigram check, the new-account check, or the LLM authenticity classifier (step 5 needs
a star rating for context that a text-only edit may not carry, and is skipped on edit to avoid an
extra DB round-trip for what's usually a minor tweak — full re-classification only happens on a
fresh review). An edit can flip a clean review to `Flagged`; it never un-flags one automatically
(an existing flag persists across edits — only staff clear it). Build the "edit review" UI to
expect this: a save can silently change the review's visibility, so re-fetch/re-render its
`status` after a successful `PATCH` and show a "your edit is pending review" notice if it comes
back `Flagged`.

If you build a staff-facing "flagged reviews" queue, source it from `Report`s (`target_type: "review"`,
`status: Open`) via whatever staff review-report listing you add, plus reviews with
`status === Flagged`; there isn't a single dedicated "list flagged reviews" endpoint beyond fetching
per-business/project reviews with `all`-style staff visibility — reports (`POST /reviews/{id}/report`)
are the user-driven flagging path exposed today.

### 8.5 Photos
Both `ReviewCreate.photos` (max 6) and `BusinessCreate/Update.gallery` (max 12) are **URL-only
fields** — this API does not accept file uploads or multipart form data anywhere. The frontend is
responsible for uploading images to its own storage (S3, Cloudinary, etc.) first and submitting
the resulting HTTPS URLs. `gallery` URLs are validated server-side as HTTPS-only; do the same
client-side validation before submit to fail fast.

---

## 9. Webhooks

Lets a business/project owner (or staff) get server-to-server push notifications instead of
polling. Build this as a "Developer" or "Integrations" settings tab scoped to one business/project
at a time (`target_type` + `target_id` query/body params throughout).

**Event catalog** (`GET /webhook-events`):

| Event | Fires when |
|---|---|
| `review.created` | A new review is posted |
| `review.updated` | A review is edited |
| `review.deleted` | A review is deleted |
| `review.voted` | A review is marked helpful/unhelpful |
| `review.responded` | The owner replies to a review |
| `queue.claimed` | Claimed for staff review |
| `queue.unclaimed` | Released back to pending |
| `queue.approved` | Approved and published |
| `queue.denied` | Denied |
| `webhook.test` | Only ever sent by the "Test Webhook" button |

**Creating one:**
```ts
POST /webhooks
{ target_type: "business" | "project", target_id: UUID, url: "https://...", events?: string[] }
→ WebhookRevealed   // includes `secret` — show it once in a copyable code block with a
                     // "you won't see this again" warning, exactly like a GitHub/Stripe secret UI
```
Leaving `events` empty/omitted subscribes to every event.

**Verifying deliveries** (documentation for whoever implements the *receiving* end, e.g. if your
frontend also ships example code for integrators): each delivery is HMAC-SHA256 of the raw request
body using the webhook's secret, sent as the `X-Metro-Signature` header. Show this in an
"Integration guide" panel next to the secret.

**Rotating a secret** invalidates the old one immediately — warn users their existing receiver
must be updated before rotating, and again show the new `secret` exactly once.

**Health**: `failure_count` (consecutive failures) and `last_triggered_at` are returned on every
`GET`/list — surface these in the webhook list UI (e.g. a red badge if `failure_count > 0`, "never
delivered" if `last_triggered_at` is null) so integrators notice broken endpoints without digging.
There's an unspecified-but-implied auto-disable threshold — if `enabled` flips to `false` on its
own, tell the user their webhook was auto-disabled after repeated failures and let them re-enable
it via `PATCH { enabled: true }` once fixed.

---

## 10. Embeddable widget

`GET /businesses/{id}/widget` is deliberately public, unauthenticated, and cacheable (5 min
server-side cache) — it's meant to be embedded on a *business's own external website*, not just
consumed by the main frontend.

```ts
{
  name: string; slug: string; avg_rating: number; review_count: number; featured: boolean;
  recent_reviews: { rating: number; title: string | null; body: string; created_at: ISODateTime }[];  // up to 3
}
```

Ship this as a small, dependency-free embed: either
1. a `<script src=".../widget.js" data-business-id="...">` that fetches this endpoint and renders
   a compact card, or
2. an `<iframe>` pointing at a tiny frontend route (e.g. `/embed/business/{id}`) that renders the
   same data server-side/statically.

Give business owners a "copy embed code" box on their dashboard (using their own `business.id`)
that emits whichever of the two you build.

---

## 11. Sponsorship / "Featured" placement

`PATCH /businesses/{id}/feature { featured: boolean, until?: ISODateTime }` is **staff-only**
(`businesses.feature` permission) and is explicitly **data-model only — there is no payment
processing anywhere in this API.** If the product needs paid sponsorship, that billing integration
(Stripe, etc.) is entirely a frontend/separate-service concern that, on successful payment, calls
this endpoint server-side (never trust the browser to call it directly with real money on the
line — do it from your own backend-for-frontend or a trusted service role, since only staff
sessions/permissions can call it).

On the frontend, `featured: true` businesses should:
- Sort first in every list/search view server-side already (no client sort needed) — see §4.3.
- Get a visual "Sponsored"/"Featured" badge.
- Optionally be surfaced in a distinct "Featured" carousel/rail on the homepage, sourced from
  `GET /businesses?sort=rating` (or any listing) filtered client-side to `featured === true`, or
  simply rely on the natural featured-first ordering and label the first N rows.
`featured_until` is nullable (indefinite featuring) — if set, consider a countdown or "featured
until {date}" label for staff managing placements.

---

## 12. Discovery & personalization features

- **Similar businesses** (`GET /businesses/{id}/similar`): render as a "You might also like" rail
  on the business detail page. Same-category, best-rated-first, max 6, always excludes the
  business itself.
- **Recommended for me** (`GET /me/recommended`, auth required): personalized homepage rail based
  on categories the user has already reviewed in. New users with no review history get overall
  top-rated businesses instead — no empty state needed, the endpoint always returns something
  (except on an empty database).
- **"Near me" search**: pass `lat`/`lng` (from `navigator.geolocation` with user permission) into
  `GET /businesses`, optionally `radius_km` to bound it, and `sort=distance`. Show `distance_km`
  (only present when lat/lng were supplied) next to each result, e.g. "2.3 km away."

---

## 13. Error handling & UX conventions

- **Every** error body is `{ message: string, error: true, context?: {...} }` — build one shared
  API-error type and one shared toast/inline-error renderer around `message`.
- Common status codes actually used: `400` (bad request/validation), `401` (missing/invalid/expired
  session), `403` (banned, wrong owner, missing permission), `404` (not found — also used to hide
  the *existence* of non-approved businesses/projects from non-staff, so a 404 doesn't always mean
  "never existed"), `409` (conflict — duplicate email on register/set-password, duplicate slug on
  business create, duplicate review on a subject the user already reviewed), `429` (rate limited),
  `500` (internal — show a generic retry message, don't parse `message` for logic).
- **Rate limits** the frontend should design around (bucket → limit): `auth-login` 20/hr,
  `auth-register` 5/hr, `auth-login-password` 10/15min, `invite-create` 30/hr, `invite-lookup`
  60/hr, `report-create` 20/hr. A 429 has no `Retry-After` body field guaranteed in the JSON, but
  does set standard rate-limit headers — read those if you want an exact countdown, otherwise show
  a generic "slow down, try again shortly."
  Note: **all rate limiting is a no-op if the server has no Redis configured** — don't build
  client-side logic that *depends* on 429s always being possible; treat them purely as a UX nicety
  to handle gracefully if they occur, never as a security boundary.
- **Pagination** is `offset`/`limit` (not cursor-based) everywhere it appears (business/project
  reviews, search, actions log). Defaults and caps vary by endpoint — see §4 tables. Build one
  shared paginated-list hook/component parameterized by `(defaultLimit, maxLimit)`.
- **PATCH responses**: business/project updates return `{ has_updated: string[] }`, *not* the
  updated entity — re-fetch the entity (or optimistically merge your own request body into local
  state) after a successful PATCH rather than expecting the response to contain fresh data. Review
  PATCH is the exception — `PATCH /reviews/{id}` does return the full updated `Review`.

---

## 14. Suggested page/route → endpoint map

A concrete starting information architecture for the public frontend:

| Page | Primary endpoint(s) |
|---|---|
| Home | `GET /businesses?sort=rating` (or `new`) for a features rail + `GET /categories` for category tiles + `GET /me/recommended` if logged in |
| Browse / category page | `GET /businesses?category=...&sort=...&offset=...` |
| Search results | `GET /search?q=...` (mixed businesses+projects) — or `GET /businesses?q=...&...filters` for a filtered directory search |
| Business detail | `GET /businesses/{slug}`, `GET /businesses/{id}/reviews`, `GET /businesses/{id}/projects`, `GET /businesses/{id}/similar` |
| Project detail | `GET /projects/{id}`, `GET /projects/{id}/reviews` |
| Write a review | `POST /reviews` (from a business or project page's "Write a review" CTA; pre-fill `invite_token` if arrived via `/invite/{token}`) |
| Invite landing page | `GET /invites/{token}` → pre-filled review form |
| Login / Register | `/auth/login` (Discord), `/auth/register`, `/auth/login/password` |
| Account settings | `GET /me` (profile), `POST /auth/password` (link a password) |
| My reviews | `GET /me/reviews` — every review the caller has written, including ones still `Flagged`, with edit/delete acting on the individual review via `PATCH`/`DELETE /reviews/{id}` |
| Claim this business | `POST /businesses/{id}/claim`, tracked afterward via `GET /me/claims` |
| Owner dashboard → My Businesses (picker) | `GET /me/businesses` — every business the caller owns or submitted, any status; use this to build the business-switcher/landing list before drilling into a single business's dashboard below |
| Owner dashboard → Overview | `GET /businesses/{id}/analytics` |
| Owner dashboard → Edit listing | `PATCH /businesses/{id}` |
| Owner dashboard → Projects | `POST/PATCH /businesses/{business_id}/projects`, `/projects/{id}` |
| Owner dashboard → Reviews | `GET /businesses/{id}/reviews`, `POST /reviews/{id}/response` |
| Owner dashboard → Invite reviewers | `POST /businesses/{id}/invites` |
| Owner dashboard → Integrations | `GET/POST/PATCH/DELETE /webhooks`, `.../rotate-secret`, `.../test` |
| Embed snippet | `GET /businesses/{id}/widget` |
| Our team (public) | `GET /team` |
| Moderation log (public) | `GET /actions` |
| Staff panel → Queue | `GET /businesses?all=true`, `GET /businesses/{business_id}/projects?all=true`, `.../review/claim\|unclaim\|approve\|deny` |
| Staff panel → Ownership claims | (no all-claims listing endpoint — a claim only surfaces via the filing user's `GET /me/claims` or by looking at a specific business; approve/deny via `.../claims/{claim_id}/approve\|deny`) |
| Staff panel → Roles | `GET/POST/PATCH/DELETE /roles`, `/permissions`, `/roles/{id}/members/{user_id}`, `/roles/sync` |
| Staff panel → Feature a business | `PATCH /businesses/{id}/feature` |

Note the roles-panel and moderation-queue pages should hard-gate on `GET /me/permissions` before
even attempting to render (redirect non-staff away), since every write endpoint they call is
staff/permission-gated server-side regardless.

---

## 15. What's intentionally out of scope for this API

Don't build frontend features that assume backend support that doesn't exist:

- **No file/image upload endpoint** — URL fields only (§8.5). Bring your own object storage.
- **No payment processing** — `feature` is a pure toggle (§11).
- **No push/websocket layer** — everything is request/response; poll or re-fetch after actions,
  there's nothing to subscribe to client-side (webhooks are server-to-server only, §9).
- **No refresh tokens / no logout endpoint** — sessions are 30-day bearer tokens you discard
  locally (§2.1).
- **No `GET /users/{id}` public profile endpoint** — you cannot look up an arbitrary user's
  profile by ID from the frontend; `author_id` on a `Review` is just a UUID with no accompanying
  username/avatar lookup exposed. If you want to show "Jane D. left a review," you'll need to add
  such an endpoint server-side first — don't assume one exists.
- **No cursor pagination** anywhere — offset/limit only, so don't build infinite-scroll assuming
  stable cursors under concurrent writes.
