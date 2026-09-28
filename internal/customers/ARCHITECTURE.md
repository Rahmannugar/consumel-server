# Customers Architecture

The customers domain owns the lightweight customer records that an integrating
business synchronizes with Consumel. The integrating business supplies the
opaque `customer_id`; Consumel does not infer identity from email or provider
data.

Every record belongs to one project environment. The same `customer_id` may
exist independently in Sandbox and Live, while a database constraint prevents
duplicates inside one environment. Name and email are optional first-class
fields. Plan, country, and location are the only supported metadata keys and
are stored as explicit bounded columns rather than arbitrary JSON.

Public endpoints receive project and environment context only from an active
Bearer API key. Dashboard endpoints receive the project and environment
explicitly, then authorize them through the signed-in user's active
organization access. Both transports call the same customer service and
repository.

Customer lists use stable cursor pagination ordered by creation time and the
internal UUIDv7 identifier. Sandbox and Live are always separate query scopes.
