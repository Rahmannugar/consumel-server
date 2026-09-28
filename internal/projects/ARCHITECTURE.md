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
