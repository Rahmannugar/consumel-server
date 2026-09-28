# Meters Architecture

The meters domain owns the definitions of what a project measures. A stable
`meter_key` identifies one meter within a project. Each project environment
owns its own name, optional description, and prepaid, postpaid, or hybrid type
for that identity, so Sandbox and Live configuration remains isolated.

PostgreSQL enforces project-level meter-key uniqueness and one configuration
per meter in each environment. Creating a key in another environment reuses
the project identity and creates an independent environment configuration.
Creating the same key twice in one environment returns a conflict.

Environment configurations are archived rather than deleted because usage,
pricing, balances, and invoices may retain historical references. Normal meter
queries exclude archived configurations and use partial indexes with the same
predicate. The project-level identity and key remain reserved permanently.

Public endpoints receive project and environment context from an active Bearer
API key. Dashboard endpoints authorize the explicit project and environment
through the signed-in user's active organization access. Both transports call
the same meter service and repository.

Meter lists use stable cursor pagination ordered by the environment
configuration's creation time and the internal UUIDv7 meter identifier.
