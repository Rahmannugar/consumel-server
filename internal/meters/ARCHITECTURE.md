# Meters Architecture

The meters domain owns the definitions of what a project measures. A stable
`meter_key` identifies one meter within a project. The key, name, optional
description, and prepaid, postpaid, or hybrid type form one project-level
definition so the meter cannot mean something different in Sandbox and Live.

PostgreSQL enforces project-level meter-key uniqueness and one activation per
meter in each environment. Creating a matching definition in another
environment reuses the project meter. Reusing the key with different defining
fields returns a conflict, as does creating the same meter twice in one
environment.

Environment activations are archived rather than deleted because usage,
pricing, balances, and invoices may retain historical references. Normal meter
queries exclude archived activations and use partial indexes with the same
predicate. The project-level definition remains reserved permanently.

Search stores weighted PostgreSQL `tsvector` values for full terms and
normalized text for partial matching. GIN full-text and trigram indexes are
scoped by project. Environment activation remains a separate active-only join.

Public endpoints receive project and environment context from an active Bearer
API key. Dashboard endpoints authorize the explicit project and environment
through the signed-in user's active organization access. Both transports call
the same meter service and repository.

Meter lists use stable cursor pagination ordered by the environment
activation's creation time and the internal UUIDv7 meter identifier.
