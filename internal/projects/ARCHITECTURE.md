# Projects Architecture

The projects domain owns projects and their Sandbox and Live environments. Each
project and environment has an internal UUIDv7 identity.

Creating a project and both environments is one PostgreSQL transaction.
Sandbox is active immediately. Live exists separately and remains inactive
until explicitly activated.
