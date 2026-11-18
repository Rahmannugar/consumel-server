# Consumption Architecture

The consumption domain owns authoritative customer and meter balances and the
synchronous Consume transaction. Usage persistence, balance mutation,
idempotency, and the transactional outbox share one short PostgreSQL
transaction.

A balance is isolated by project environment, customer, and meter. Database
foreign keys prevent a customer or meter configuration from another
environment from being attached to the balance. Quantities are nonnegative
signed 64-bit integers.

Additive provisioning requires a UUIDv7 idempotency key scoped to the project
environment. The durable operation records the public request identity and the
original result. An exact replay returns that result, while reusing the key for
a different request returns a conflict. The claim, balance mutation, and stored
result commit in one transaction.

The balance row serializes concurrent changes, while durable entitlement grant
lots are the source of available quantity. Additions may expire. Reads exclude
expired lots without depending on scheduled cleanup, and reductions spend the
earliest-expiring lots first. Grant allocations preserve which lots funded a
usage operation or an exact downward adjustment. Exact setting is naturally
idempotent; concurrent add and set requests take effect in commit order.

Normal reads join the active environment meter configuration and therefore
exclude balances for archived meters. Historical operation records retain the
stable customer and meter identity required for investigation and replay.

Consume requires a UUIDv7 idempotency key. An accepted prepaid operation locks
and debits sufficient balance; an insufficient prepaid operation commits a
durable denial and its transactional outbox event. Postpaid usage is accepted
without a balance, while hybrid usage debits the available amount and accepts
any remainder. A missing customer is created as a lightweight
environment-scoped record so a synchronization race cannot lose usage.

Every accepted or denied operation stores the usage decision and matching
transactional outbox row. Only accepted Live operations are billable. Provider
calls, analytics, webhook delivery, and outbox publication never run inside
the synchronous transaction.

Dashboard history reads accepted and denied operations from PostgreSQL with
bounded cursor pagination and explicit date ranges. Matching lifecycle-aware
partial indexes exclude pending claims and cover unfiltered and status-filtered
history. The Events SSE channel uses Redis Streams only as a notification
transport, then reloads each authoritative operation from PostgreSQL before
delivery. Reconnection invalidates the REST history so Redis retention cannot
make the dashboard's durable history incomplete.

Bounded analytics reads aggregate finalized operations directly from PostgreSQL.
They return continuous UTC hourly or daily buckets and may be scoped to one
customer, one meter, or their intersection. Hourly ranges are limited to 31
days and daily ranges to one year. The read path is downstream of Consume and
never adds aggregation work to the atomic write transaction. A durable rollup
pipeline remains a later scale decision that must preserve this authoritative
contract and support replay and backfill before replacing raw aggregation.

## Recurring customer allowances

Consumption owns the customer-and-meter allowance schedule because it issues
quantities into the same grant ledger used by balance adjustments and Consume.
Each balance has at most one policy, isolated by project environment. Immutable
revisions retain actor and source and take effect next renewal. The original
monthly UTC anchor is preserved when a shorter month clamps its day.

Period records, grants, the balance projection, the next due date, and a
transactional outbox event commit under the balance lock. Consume, exact
adjustments, and additive provisioning use that same lock and reconcile a due
period before deciding available entitlement. The entitlement clock is captured
after the lock so a waiter cannot spend an expired grant after its quantity has
already been carried into another period. Expiration checks use that captured
instant throughout the transaction.

A zero cap resets policy-issued quantity. A positive cap carries only this
policy's unused units, in addition to the new monthly allowance. Manual grants
retain their separate expiry rules. Recovery computes capped carry across
untouched missed periods without an outage-sized loop; only the current period
gets a spendable grant. Initial creation never accrues grants for earlier
periods. The period key prevents duplicate issuance across workers and request
paths, and immutable period records retain the revision, carry, and skipped
period count needed to explain recovery.

The worker uses two independent renewal slots with bounded batches and a
15-second discovery interval. Failed renewals remain due and are delayed one
minute with a durable attempt count. A failing period transaction rolls back
its grant, projection, schedule advancement, and event together. Exact balance
reductions remain available to recover when a due grant would exceed bigint
capacity. Archived meter configurations and inactive organizations are excluded
from background renewal. No external provider is part of issuance.

This customer quantity policy does not implement Consumel organization-plan
operation allowances or billing periods. Reusable plan/segment policy defaults,
other schedule intervals and timezones, pricing tiers, minimum commitments,
stable pricing snapshots, reconciliation, provider activity, and broader
background-job visibility belong to their applicable later slices.
