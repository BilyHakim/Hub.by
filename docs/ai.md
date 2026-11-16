# Hubby AI

Hubby AI is available at `/ai`, with personal settings at `/ai/settings`. Each
credential, conversation, and request belongs to a user and workspace. The module
uses the application's existing login. Workspace membership is checked on every
request, and a conversation keeps its original workspace while generation runs.

## Setup

1. Run migration `028_ai.sql` using `go run ./cmd/migrate up` from `backend`.
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

Finance access starts disabled. When enabled, context includes the selected
finance period and three preceding periods, expense categories, budget versus
actual amounts, and a backend-calculated scenario for discretionary spending.
Periods follow Hubby's existing finance-period settings. Optional scopes include
aggregate balances, goals, and outstanding debt/receivables. These are current
snapshots, not historical account balances. Expense includes emergency spending;
transfers are excluded. A finished period does not guarantee complete records.

The reduction scenario applies the chosen percentage to recorded discretionary
expenses, rounded down per category. It shows hypothetical savings and surplus
for that period. The AI has no transaction/budget write tools, free-form SQL, or
database credentials. Finance reads run in a read-only repeatable-read transaction
with query timeouts and explicit workspace filters. Raw transaction descriptions,
account numbers, and notes are omitted. Category/goal names can still be sensitive.

The provider receives the question, enabled context, and up to six recent turns
within a 48 KB history budget. If source permissions narrow, incompatible prior
history is omitted from the provider request. Outputs are displayed as escaped
text. Calculated context accompanies every answer for review; generated advice
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
requests older than three minutes are released on the next request. A crash after
provider acceptance can incur usage without a saved answer; no automatic inference
retry is performed. Token totals describe successfully saved provider responses,
not an authoritative billing ledger. Database failures after generation may also
prevent saving an otherwise billable response.

## Validation

Run `go test ./...` in `backend` and `npm.cmd run build` in `frontend` on Windows.
For the AI database lifecycle test, set `AI_TEST_DATABASE_URL` to a disposable
PostgreSQL database migrated through 028 before running Go tests. The test uses a
local mock provider and covers encryption, data scopes, simulation, replay, and
cross-user/workspace access without sending real financial data to an AI provider.
It also checks that the actual Finance read transaction rejects INSERT, UPDATE,
DELETE, TRUNCATE, and DROP with PostgreSQL error 25006. A comparison of all Finance
source rows before and after analysis and AI cleanup verifies that source data
stays unchanged, including when provider output contains SQL and a tool-call attempt.
