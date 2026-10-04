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

This domain currently implements the atomic quantity operation and the balance
behavior associated with prepaid, postpaid, and hybrid meters. It does not
claim to complete Consumel's broader metering product. Recurring
allowance policies, scheduled resets, rollover, customer overrides, pricing tiers,
minimum commitments, billing periods, stable pricing snapshots,
reconciliation, provider activity, and background-job visibility remain owned
by their applicable metering, pricing, billing, reconciliation, connector, and
operational slices.
