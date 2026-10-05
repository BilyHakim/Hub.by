# Hubby Maintenance: implementation and verification

Date: 2026-10-05. The implementation is present in the working tree. Browser acceptance remains pending, so the UI is not yet marked ready for release.

## Changed files

| Files | Purpose |
| --- | --- |
| `backend/migrations/025_maintenance.sql` | Workspace-scoped categories, items, rules, and immutable service history; composite foreign keys, indexes, and Goose rollback. |
| `backend/internal/httpapi/maintenance.go` | Validation, CRUD, schedule calculation, usage reminders, and transactional completion. |
| `backend/internal/httpapi/maintenance_test.go` | Calendar boundaries, status thresholds, combined intervals, and validation tests. |
| `backend/internal/httpapi/maintenance_integration_test.go` | PostgreSQL lifecycle tests, historical snapshots, usage validation, and workspace isolation. |
| `backend/internal/httpapi/router.go` | Seven protected Maintenance endpoints using the existing auth and request middleware. |
| `frontend/src/views/MaintenanceView.vue` | Dashboard, searchable item list, detail, history, and item/rule/completion forms. |
| `frontend/src/components/MaintenanceDialog.vue` | Module dialog wrapper with focus restoration, Tab containment, Escape, and busy-state handling. |
| `frontend/src/utils/maintenance.js`, `maintenance.test.js` | Shared age, formatting, status, and optional-number helpers and boundary tests. |
| `frontend/src/services/api.js`, `frontend/src/router.js` | Existing request wrapper and four Maintenance routes. |
| `frontend/src/App.vue`, `frontend/src/views/HubView.vue` | Module card, sidebar navigation, product title, and working Maintenance header search. |
| `README.md`, this file | Setup, behavior, verification, and remaining acceptance checks. |

## Design decisions

- Reference: Books supplies the list/detail/history and modal patterns; the existing shell supplies workspace navigation.
- Palette: sage, sand, rose, and neutral tokens communicate normal, upcoming, overdue, and unknown states in the existing Hub.By vocabulary.
- Typography: inherited DM Sans and Manrope preserve the application's visual identity.
- Layout and spacing: existing page, panel, form-grid, button, and responsive patterns keep the new module consistent; summary, attention, and inventory have distinct content roles.
- Cards: reuse MetricCard for actual database-derived counts and panels for schedules and inventory.
- Icons: reuse the installed Lucide library; Wrench denotes maintenance, CalendarDays denotes schedules, and History denotes service records.
- Motion: retain existing button/modal/toast transitions to acknowledge interaction; no new decorative animation.
- Design Read: personal asset maintenance in Hub.By's visual language; ENERGY 1 / RHYTHM 2 / MOTION 1.
- Existing Finance categories represent income/expense, so Maintenance uses a separate workspace-scoped category table with free-form names.
- A module dialog wrapper is needed because existing modals are inline forms rather than reusable dialog components; shared components are not changed.

## Verified

| Check | Result and evidence |
| --- | --- |
| Frontend production build | PASS: `npm.cmd run build`, Vite production build succeeded. |
| Frontend helper tests | PASS: `node --test src/utils/*.test.js`, all five tests passed, including existing calculator tests. |
| Backend suite | PASS: `go test ./...`, all packages passed; the PostgreSQL integration test was enabled. |
| Go static checks | PASS: `go vet ./...`, no findings. |
| Migration | PASS: all migrations applied on an isolated PostgreSQL 16 database; migration 025 down/up succeeded. |
| API/database lifecycle | PASS: empty overview, optional fields, custom categories, item/rule edits, completed services, preserved historical names, historical date ordering, usage reminders, invalid requests, cross-workspace rejection, and item deletion cascade. |
| Vue initial runtime | PASS: server-side render of all four Maintenance routes completed with no warnings. This checks initial rendering only. |
| New status and text contrast | PASS: muted text 6.10:1; upcoming text 6.53:1; overdue text 6.44:1; good-state text 7.50:1. New input boundary color is 4.10:1 against white, exceeding the 3:1 non-text threshold. |
| Scope review | PASS: existing module views and dependencies are unchanged; shared shell edits concern Maintenance integration only; no sample asset records are seeded. |
| Whitespace | PASS: `git diff --check`. |

## Acceptance still required

No browser is exposed by the session's computer-use tools. Browser inventory was empty, and both Edge and the in-app browser reported unavailable. Server-side rendering is not a substitute for visual or browser interaction testing.

The antislop Delivery Gate remains pending for R-03, R-32, R-34/C-4, and R-35 until the following are verified in a browser:

1. Open Maintenance from the hub and navigate through summary, items, detail, and history.
2. Add/edit a minimal item and a fully populated item; check validation, image fallback, category input, and money fields.
3. Add/edit/deactivate rules; complete time-only, usage-only, and combined rules; inspect resulting history and schedules.
4. Exercise list filters, both search fields, error/retry, empty state, and save feedback.
5. Switch workspaces while requests are in flight and confirm that stale data and forms do not remain visible.
6. Check dialogs by keyboard: Tab, Shift+Tab, Enter, Escape, close button, backdrop, focus restoration, and saving states.
7. Inspect 360px, 768px, and desktop widths, plus 200% zoom, for overflow, clipping, tap targets, and readability.
8. Check the browser console and network panel for errors during the complete flow.

Before using the module on the application's database, apply migration 025 with the existing migration command. Verification used only a disposable database; the application's configured database was not migrated.
