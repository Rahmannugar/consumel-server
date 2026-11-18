# Consumption Architecture

Consumption owns customer-and-meter grant balances, recurring customer
allowances, synchronous usage decisions, and their history and analytics.
Each balance is isolated by project environment, customer, and meter, with
foreign keys enforcing those relationships and nonnegative bigint quantities.

## Grant ledger and atomic consumption

Durable grant lots are authoritative; the available balance is their active
projection. Expired lots remain historical but cannot be spent. Consumption and
exact reductions allocate the earliest-expiring lots first, using never-expiring
lots last. Allocation records explain which grants funded a debit.

The balance row serializes additions, exact adjustments, usage debits, and
allowance renewal. Additions and Consume use environment-scoped UUIDv7
idempotency keys: identical retries return the original result, while changed
requests conflict. The claim, mutation, result, and applicable outbox event
commit together.

Prepaid consumption requires sufficient balance. Postpaid consumption needs no
balance; hybrid consumption debits available units and accepts the remainder.
Accepted and denied decisions are durable, and only accepted Live operations
are billable. Providers, publication, and analytics stay outside the transaction.

## Recurring customer allowances

Each balance may have one policy. Schedules use a count of UTC minutes, hours, days, weeks, months, or years,
with a one-minute minimum. Minutes through weeks are elapsed time; months and
years preserve the calendar anchor and clamp shorter months without drifting.
Versioned revisions retain actor/source and take effect at the next renewal.
A changed interval is anchored to that renewal; unchanged intervals retain their
anchor. Existing monthly policies and omitted schedule fields retain the
one-month default.
A zero cap resets policy-issued units; a positive cap bounds unused units carried
in addition to the new grant. Manual grants retain independent expiration rules.

Renewal commits the period, grant, projection, next due date, and outbox event
under the balance lock. Period identity prevents duplicate issuance across
workers and request recovery. The entitlement clock is captured after locking,
so waiters cannot spend expired units already carried into another period.

The worker renews due policies with bounded capacity. Consume and balance
mutations also reconcile due periods without depending on worker availability.
Missed-period recovery calculates capped carry without an outage-sized loop and
makes only the current period spendable. Failures roll back issuance together;
exact reductions remain a recovery path when pending issuance would overflow.
Period history retains the applied revision, carry, and skipped-period count.

Custom UTC schedules are one connected slice, not the complete flexible
allowance capability. Local timezone schedules, subscription-period alignment,
and reusable plan/segment defaults remain undelivered V1 work. Consumel organization-plan operation allowances and billing
periods belong to separate organization billing state.

## Reads and operational visibility

Normal reads exclude archived meter configurations. PostgreSQL supplies
cursor-paginated grant, balance, period, and usage history. Redis Streams supplies
usage notifications; SSE reloads authoritative operations and reconnection
refreshes REST history so stream retention cannot erase durable history.

Bounded analytics aggregate finalized PostgreSQL operations by UTC hour or day,
optionally scoped to a customer or meter. Aggregation stays outside Consume;
future durable rollups must preserve the authoritative contract and support
replay and backfill before replacing direct reads.
