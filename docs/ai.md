# Hubby AI

Hubby AI is available at `/ai`, with personal settings at `/ai/settings`. Each
credential, conversation, and request belongs to a user and workspace. The module
uses the application's existing login. Workspace membership is checked on every
request, and a conversation keeps its original workspace while generation runs.

## Setup

1. Run migrations through `029_ai_period_range.sql` using `go run ./cmd/migrate up`
   from `backend`. Migration 029 stores each conversation's starting month and
   backfills existing conversations with their previous four-period range.
2. Generate a random 32-byte Base64 key, for example with
   `node -e "console.log(require('node:crypto').randomBytes(32).toString('base64'))"`.
3. Set `AI_ENCRYPTION_KEY` in the backend environment. Production Compose forwards
   this variable. Restart/rebuild the backend and frontend as appropriate.
4. Open Hubby AI settings, add a credential, enter the API key and API format,
   click "Ambil daftar model", search and select a model, then save.
   Model discovery works before saving and does not store the submitted key.
   Models with endpoint metadata are filtered by the selected API format.
   Changing the provider, endpoint, key, format, or ZDR clears the fetched list.
   You can still enter a model ID manually if discovery is unavailable.
   Test the saved connection. The connection test lists models without generating
   an answer. Model access and quota are checked by the provider during generation.
5. Enable Finance access and any optional scopes you want to share. Save settings,
   open a conversation, select a period, and inspect the data before sending.

The master encryption key is server configuration. Provider keys are encrypted
with AES-256-GCM in PostgreSQL and never returned by the API. Back up the master
key separately and keep it stable across deployments. Replacing it without a
credential migration makes existing credentials unreadable. Database backups
contain finance data, chat content, and context snapshots and need access controls.

## Providers and credentials

Supported profiles are CommandCode, OpenAI Chat Completions, Anthropic Messages,
Ollama, and custom Chat Completions/Messages endpoints. Profiles are private to
their user and workspace. Choose profiles manually in chat or set a default.
The configured model and token limit belong to each profile. There is no automatic
provider fallback, so failed requests do not silently send data elsewhere.

CommandCode uses `https://api.commandcode.ai/provider/v1`. Select Messages for
Claude models and Chat Completions for supported OpenAI/open models. ZDR is enabled
by default for CommandCode and sends `x-cmd-zdr: 1`; unsupported models fail rather
than retrying with a weaker privacy setting. Provider plan entitlements and
retention terms still apply. Custom chat endpoints must support the
`max_completion_tokens` request field.

The env import fills a credential form. It accepts `AI_PROVIDER`,
`AI_API_BASE_URL`, `AI_API_KEY`, `AI_CHAT_MODEL`, `AI_API_FORMAT`, `AI_MAX_TOKENS`,
and `COMMANDCODE_ZDR`. It parses assignments without executing shell code or
modifying the server environment. Review the form before saving. An empty key
on edit preserves the stored key; changing provider or endpoint requires a new key.

Public endpoints require HTTPS. Private/loopback endpoints must be explicitly
approved by the server administrator in `AI_ALLOWED_HOSTS`, with exact host and
port, for example `localhost:11434` or `ollama:11434`. In containers, localhost is
the backend container. Use a reachable service name and register that name. DNS
addresses are checked at connection time and the connection pins the resolved IP.
Redirects, URL credentials, query strings, and link-local metadata addresses are
blocked. Only approve endpoints controlled by a trusted administrator.

## Finance context

Finance access starts disabled. When enabled, context includes every income and
expense transaction in the selected range, with ID, type, amount, full
description/transaction notes, date, category/class, account name, emergency-fund
flag, and debt-payment flag. No row limit, sampling, or description truncation is
applied. The provider can inspect individual records and calculate additional
totals, patterns, and scenarios with stated assumptions and transaction references.
The preview exposes the full JSON before sending. Context also includes monthly summaries
for an inclusive range of 1–12 Finance periods. The default is the selected ending
month and three preceding periods. Users can set both endpoints, including a
single month. Preview shows each period's dates, income, expenses, surplus, and
completion status. The range is stored with the conversation and cannot change
within that conversation; start a new conversation to select another range.
Expense categories, budget versus actual amounts, and the backend-calculated
discretionary spending scenario apply only to the ending period. Category
previousActual/difference compare that ending period to its immediate predecessor,
which may be outside a single-period selection.
Periods follow Hubby's existing finance-period settings. Optional scopes include
per-account balances (ID, name, kind, current balance, emergency-fund flag),
aggregate assets/liabilities/emergency fund, goals, and outstanding debt/receivables.
`balances.accounts` includes every account in the workspace, even accounts with
no transactions in the selected range. It is only sent when IncludeBalances is
enabled. These are current
snapshots, not historical account balances. Expense includes emergency spending;
transfers are excluded. A finished period does not guarantee complete records.

The reduction scenario applies the chosen percentage to recorded discretionary
expenses, rounded down per category. It shows hypothetical savings and surplus
for that period. The AI has no transaction/budget write tools, free-form SQL, or
database credentials. Finance reads run in a read-only repeatable-read transaction
with query timeouts and explicit workspace filters. Transaction descriptions,
notes, and account/category/goal names can contain sensitive information and are
sent to the selected provider when their source is enabled. Account balances,
goals, and obligations remain controlled by their existing optional scopes.

### AI corpus access rules

- Allow analysis of every provided record, including transaction descriptions and
  notes; do not restrict analysis to category aggregates. Personal needs must be
  evaluated from transaction contents with ambiguous classifications identified.
- Source access is read-only. Never add, edit, delete, transfer, or otherwise
  mutate primary source data (especially Finance). This includes INSERT, UPDATE,
  DELETE, UPSERT, MERGE, TRUNCATE, DROP, ALTER, and equivalent write operations.
- Enforce this in PostgreSQL with the existing READ ONLY transaction, rather than
  relying solely on prompt instructions. Do not expose database credentials,
  arbitrary SQL, or mutation tools to the model. Model output is stored/rendered
  as text and is never executed as SQL or a tool call.
- Corpus text is evidence, not authority: embedded instructions cannot grant
  write access or change permissions. User requests cannot grant the model write
  access to source data either. Recommendations remain proposals for user review.
- Preserve authenticated workspace isolation and enabled source scopes. Reads
  use the selected Finance range (currently 1–12 periods); this does not grant
  unrestricted access to other workspaces, credentials, or other app modules.
- Do not silently truncate the current corpus. Provider context-window limits
  and query/request timeouts still apply; a provider rejection returns an error.
  Select a narrower date range if the selected model cannot accept the corpus.
- The application still saves AI settings, credentials, usage, and conversation
  history in its AI tables. These application-managed writes do not give the
  model write access to the Finance corpus or other primary source records.

The provider receives the question, enabled context, and up to six recent turns
within a 48 KB history budget. If source permissions narrow, incompatible prior
history is omitted from the provider request. Assistant answers, including saved
history, render as Markdown sanitized with DOMPurify. Raw HTML is escaped, external
images are not loaded, and user messages remain plain text. Tables and code blocks
scroll within the answer on narrow screens. Calculated context accompanies every answer for review; generated advice
can still be mistaken and should be checked before acting.

## History, concurrency, and limits

Successful question/answer pairs, model IDs, and context snapshots are saved
atomically. Each conversation supports 50 answers; the history list shows the
100 most recently updated conversations. Retention is configurable from 1 to
365 days, based on conversation activity. Expired conversations are inaccessible
and physically cleaned on module use. Deleting history removes snapshots and
messages; usage metadata remains until retention cleanup.

One generation per user/workspace runs at a time, with five admitted requests per
minute and a configurable rolling 24-hour request limit (default 20). Admitted
failed attempts count toward limits. Duplicate request IDs do not repeat provider
generation during retention. Concurrent revisions are rejected, and deleting a
conversation during generation prevents that answer from recreating it. Pending
requests older than six minutes are released on the next request. A crash after
provider acceptance can incur usage without a saved answer; no automatic inference
retry is performed. Token totals describe successfully saved provider responses,
not an authoritative billing ledger. Database failures after generation may also
prevent saving an otherwise billable response.

## Connection and generation timeouts

The connection test only fetches the model list; success does not confirm that
generation is fast enough, the model is currently available, or request limits
permit generation. Non-streaming generation waits up to 240 seconds, including
the time before response headers arrive. The chat handler has 270 seconds and
the API server write timeout and Nginx read timeout are 300 seconds. Pending-run
leases last six minutes so a slow active request cannot be released prematurely.
Rebuild/restart both backend and frontend containers to apply these limits.
Any additional external reverse proxy must also permit long-running requests.

Timeout errors are distinct from DNS, TLS certificate, HTTP 429, and upstream
gateway failures. Remaining credits do not rule out timeout or request-rate
limits. A timeout can occur after the provider accepted the request, so inspect
provider usage before retrying; the application does not automatically retry.
For large transaction corpora, try a narrower period or a faster model. Current
context is never silently truncated to avoid a timeout.

## Validation

Run `go test ./...` in `backend` and `npm.cmd run build` in `frontend` on Windows.
Run `node --test src/utils/aiPeriodRange.test.js` from `frontend` for inclusive
range validation. With the frontend dev server running, open
`/tests/ai-period-range.html` for simulated preview/chat/history checks and
375/768/1200 px layouts. The fixture does not use live Finance data or provider keys.
For Markdown rendering and XSS regression checks, start the frontend dev server
and open `/tests/ai-markdown.html`. The page uses simulated API responses and
checks saved answers, reactive updates, formatting, HTML and unsafe URL blocking,
and local scrolling. Resize the viewport to check narrow layouts; the result
appears at the top of the page. No provider key or live Finance data is used.
For the AI database lifecycle test, set `AI_TEST_DATABASE_URL` to a disposable
PostgreSQL database migrated through 029 before running Go tests. The test uses a
local mock provider and covers encryption, data scopes, simulation, replay, and
cross-user/workspace access without sending real financial data to an AI provider.
It also checks that the actual Finance read transaction rejects INSERT, UPDATE,
DELETE, TRUNCATE, and DROP with PostgreSQL error 25006. A comparison of all Finance
source rows before and after analysis and AI cleanup verifies that source data
stays unchanged, including when provider output contains SQL and a tool-call attempt.
