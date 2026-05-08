# Goalrail Console Web

Real Goalrail console shell.

Canonical source: `apps/web/console`.

Current scope:
- public `/start` route for English global entry traffic; it uses local quick
  questions and static answers as fallback, and can submit short public
  questions to same-origin `POST /api/start-chat` when the separate start
  assistant Worker is routed; no browser OpenAI key, repo scan, code execution,
  analytics, cookies, sessions, uploads, CRM, or chat history
- multilingual EN/RU source using `react-i18next` + `i18next`
- login-only entry screen with no registration path
- auth client for the existing `apps/server` auth API:
  `POST /v1/auth/login`, optional `POST /v1/auth/change-password`,
  `GET /v1/me`, and `POST /v1/auth/logout`
- first-login password-change flow for bootstrapped users with
  `must_change_password = true`
- authenticated shell entry only after `/v1/me` succeeds
- access and refresh tokens are held in React memory only
- left navigation with Contracts, Delivery Readiness, and Proof product
  surfaces; Contracts consumes authenticated, organization-scoped, read-only
  Contract endpoints: `GET /v1/contracts`, `GET /v1/contracts/{id}`, and
  `GET /v1/contracts/{id}/current-draft`; it renders a contract rail/list plus
  selected aggregate detail and current draft body when available;
  Delivery Readiness consumes read-only `GET /v1/qualification-feed`
  while authenticated and renders Qualification / Clarification / Contract /
  Blocked lanes
- bottom-left Settings utility with Appearance theme presets and API-backed
  Organization Users add/edit/temporary-password reset UI plus read-only
  Repository context metadata
- Settings / Users blocks self-demotion from owner, self membership
  deactivation, and self temporary-password reset; own password changes use
  the existing password-change flow
- selected theme persists only as a local browser visual preference under `goalrail.console.theme`
- locale is not persisted in browser storage; runtime switching updates i18next,
  `document.documentElement.lang`, and the URL `lng` query param
- `VITE_GOALRAIL_API_BASE_URL` configures the API base URL at build time; empty
  means same-origin `/v1/...`
- no cookies, token `localStorage` / `sessionStorage`, public registration,
  signup, SSO, invite/reset email, self-service password reset, password reset
  email delivery, analytics,
  repo integration, runner, gate, proof, or product data loop
- live `console.goalrail.ru` may still point to an older static release until a
  separate deployment migration / API routing slice
- live `https://goalrail.dev/start` is served from this app through the main
  console deployment, while same-origin `POST /api/start-chat` is owned by the
  separate Cloudflare Worker, not by `apps/server`; external static serving
  must preserve SPA fallback for `/start`
- live `https://goalrail.dev/` is a temporary public root entry that redirects
  client-side to `/start` until a separate public landing page exists
- local Vite dev proxies `/api/start-chat` to `https://goalrail.dev` by default
  so the public assistant works without a local Worker; set
  `START_ASSISTANT_PROXY_TARGET=http://127.0.0.1:8787` when intentionally
  testing a local start-assistant Worker

Delivery rule:
- CLI and server functionality should become real first
- console cards and detail views should appear only after the underlying functionality exists
- Settings / Users uses `/v1/me` to determine the current `organization_id`
  and then consumes the ADR-0027 Organization user-management routes:
  `GET /v1/organizations/{organization_id}/users`,
  `POST /v1/organizations/{organization_id}/users`,
  `PATCH /v1/organizations/{organization_id}/users/{user_id}`, and
  `POST /v1/organizations/{organization_id}/users/{user_id}/temporary-password-resets`
- Settings / Repository uses `/v1/me` to determine the current
  `organization_id` and then consumes
  `GET /v1/organizations/{organization_id}/repository-context`
- Delivery Readiness polls `GET /v1/qualification-feed?limit=50` only while the
  authenticated surface is open; it renders stored feed snapshots and does not
  call Goal continuation, readiness recomputation, clarification creation, or
  contract creation automatically; open clarification questions are shown as
  read-only backend state
- Delivery Readiness exposes no `continue_goal`, `answer_clarification`, or
  `draft_contract` controls; linked contract cards expose `Open contract`
  navigation only, which loads the selected contract through
  `GET /v1/contracts/{id}`
- The Contracts surface consumes authenticated, organization-scoped read-only
  Contract discovery at
  `GET /v1/contracts?project_id=&repo_binding_id=&goal_id=&state=&limit=`,
  loads `GET /v1/contracts?limit=50` by default, supports a state filter and
  manual refresh, keeps manual ID lookup as a secondary fallback, and shows
  selected detail through authenticated, organization-scoped, read-only
  `GET /v1/contracts/{id}` as the compact public Contract aggregate only
- Repository context data is server-owned metadata only: Organization summary,
  active Project metadata, and active RepoBinding metadata
- Repository context does not imply provider authorization, checkout
  permission, readiness/proof status, execution status, or runner state
- Selected Contract detail presents public Contract aggregate fields:
  `id`, `repo_binding_id`, `goal_id`, lifecycle `state`, current lifecycle
  record ids when present, calm created/updated timestamps, and the linked
  current draft body when `current_draft_id` exists and
  `GET /v1/contracts/{id}/current-draft` is available. It does not show task
  plans, execution evidence, gate decisions, proof artifacts, runner state,
  stage controls, execution/proof meters, or fake downstream body sections.
- Selected Contract detail access errors from `GET /v1/contracts/{id}` are
  surfaced as contract-specific messages for missing, unauthorized, forbidden,
  membership, server-not-ready, network, parse, and status-coded server
  failures.
- Users data is loaded from the server API; local state is only the fetched
  view, filters, form draft, and one-time create/reset response panel
- Users persistence uses backend-aligned roles only:
  `owner`, `admin`, `member`, and `viewer`; `observer` is not a documented
  target role
- generated temporary passwords are shown only from the immediate successful
  create/reset response and must not be stored in browser storage
- user management remains Console/admin API-backed; there are no CLI
  user-management commands
- product surfaces, auth state, locale, users, and settings screen are not persisted

## Goal / Contract workflow boundary

Agreed production direction:

```text
Agent -> Goalrail CLI -> Goalrail Server canonical state -> Console read-only dashboard
```

For Goal and Contract workflow, the Console is a read-only Intent & Oversight
surface. Workflow mutations happen through the local agent by calling Goalrail
CLI. The server owns canonical state. The Console visualizes server-owned
state; it must not become a second workflow driver.

The current Delivery Readiness implementation follows this boundary: it renders
server-owned feed state, read-only clarification questions, readiness / blocked
/ linked-contract state, and linked-contract navigation without Goal or
Contract workflow mutation controls.

Production dashboard must not expose controls for:
- `POST /v1/goals/{id}/continuation`
- `POST /v1/clarifications/{id}/answers/continuation`
- `POST /v1/contracts`
- contract submit / approve / plan mutation actions, if present

Allowed read-only or navigation actions:
- Open contract
- View details
- Refresh
- Select contract
- Filter
- Search

Do not add `Managed via CLI` labels or `Copy CLI command` buttons. The Console
should make the boundary clear through behavior, not by adding command-helper
chrome.

## Status refresh

The current implementation uses the simplest frontend refresh path:
- initial refresh when the user is authenticated
- frontend periodic calls to read-only backend endpoints
- repeat about every 5-10 seconds while the tab is active
- skip scheduled refresh when `document.hidden` is true
- keep manual Retry / Refresh as fallback
- keep existing visible state on transient read errors

Minimum read-only endpoint:

```http
GET /v1/qualification-feed?limit=50
GET /v1/contracts?limit=50
```

When a selected contract is open:

```http
GET /v1/contracts/{id}
GET /v1/contracts/{id}/current-draft
```

All selected Contract read endpoints are bearer-authenticated,
organization-scoped, and read-only. The current-draft endpoint is consumed only
for the selected Contract's current draft body. If the selected Contract has no
`current_draft_id`, the frontend does not call this endpoint and shows that no
current draft is linked yet.

Backend read-only contract discovery for the Contracts rail/list:

```http
GET /v1/contracts?project_id=&repo_binding_id=&goal_id=&state=&limit=
```

This is simple periodic polling. It is not true long polling, server
wait/cursor semantics, SSE, WebSocket, a daemon, or an event stream.

## Dashboard behavior

- A Goal created through CLI appears in Delivery Readiness after Console
  refresh/polling.
- A Contract created or updated through CLI appears or updates in the Console
  after refresh/polling.
- Linked contract state is visible without manual contract ID loading as the
  main user flow.
- Delivery Readiness shows qualification state and handoff to Contracts, not
  lifecycle controls.
- The Contracts surface is read-only, lists contracts from discovery, supports
  state filtering plus manual refresh, and can show selected detail or a manual
  ID lookup result.

D-0091 display behavior:
- Delivery Readiness cards show one frontend-projected primary status instead
  of duplicate readiness / next-action status chips.
- Delivery Readiness cards use calm browser-local timestamp labels and avoid
  seconds, raw ISO/RFC3339 strings, or timezone-heavy values on normal cards.

Readiness primary statuses, in display priority:
1. Needs answer
2. Ready for contract
3. Needs qualification
4. Contract linked
5. Blocked

Contract primary statuses:
- Draft
- Ready for approval
- Approved
- Blocked
- Superseded

Normal timestamp examples:
- just now
- 5 min ago
- 2 h ago
- Today 14:20
- Yesterday 09:10
- 8 May

Linked contract handoff:
- when a feed item has a linked contract summary or id, show `Contract linked`
- show compact contract id / title / state if available
- offer `Open contract` navigation only
- do not show `Draft contract` or other mutation actions
- selected contract detail loads through read-only `GET /v1/contracts/{id}` and
  renders the current draft body through read-only
  `GET /v1/contracts/{id}/current-draft` when `current_draft_id` is present

Backend discovery used by the frontend list/rail:

```http
GET /v1/contracts?project_id=&repo_binding_id=&goal_id=&state=&limit=
```

That endpoint is authenticated, organization-scoped by active membership,
read-only, compact, and limit-only. It does not recompute readiness, create
contracts, or perform lifecycle transitions. Prefer filtered
`GET /v1/contracts?goal_id=` before adding `GET /v1/goals/{goal_id}/contract`.

Deferred ideas:
- A future daemon/status heartbeat may publish lightweight agent/runtime status,
  enabling an honest `Agent working` UI state. This is not part of the current
  slice because there is no daemon or heartbeat source of truth yet.
- Activity timeline / agent-run history is deferred.
- UI clarification answer forms are deferred.
- Proof/readiness data must not be faked.
