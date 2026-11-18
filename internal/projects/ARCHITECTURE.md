# Projects Architecture

The projects domain owns projects and their Sandbox and Live environments. Each
project and environment has an internal UUIDv7 identity.

Creating a project and both environments is one PostgreSQL transaction.
Sandbox is active immediately. Live exists separately and remains inactive
until explicitly activated.

The projects domain also owns each environment's integration credential. V1
allows one active API key per environment. Creation and replacement lock the
environment row, and a partial unique index protects the same invariant under
concurrency. Consumel stores a SHA-256 digest of the high-entropy key plus safe
display metadata. The plaintext exists only in the immediate create or replace
response and is never persisted.

Public API authentication hashes the presented Bearer key and resolves only an
active key whose project environment and organization remain active. Successful
resolution establishes organization, project, and environment context without
requiring those identifiers in the public request. `last_used_at` is refreshed
at a bounded cadence so key metadata remains useful without adding a write to
every request.

## Project portfolio

`GET /v1/projects/portfolio` reads every project in the caller's active
organizations for one environment. It returns no project when the caller is not
an active member of the organization, and Sandbox and Live are never combined in
one response.

The read uses two queries. The project list reads only project and environment
rows. The usage aggregate groups finalized operations by project and calendar
bucket, and the service derives both the organization chart buckets and each
project's comparison totals from those rows. Deriving both views from one
operation scan keeps the portfolio within one pass over the usage table instead
of aggregating it once per view. Projects without operations remain in the list
with zero totals and no last activity.

Buckets stay continuous in UTC. Missing periods are returned as zero values,
hourly requests are capped at 31 days, and daily requests are capped at one
year. The response contract stays stable if raw aggregation is later replaced by
durable rollups.
