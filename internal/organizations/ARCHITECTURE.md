# Organizations Architecture

The organizations domain owns organizations and organization memberships.
The users domain separately owns user identities and their Clerk identity links.

Organization memberships connect users to organizations and carry the
predefined tenant role and access status. Creating an organization and its
active owner membership is one PostgreSQL transaction so neither record can
exist without the other.

Ownership is recorded on the organization through `owner_user_id`. Admin and
Developer are the assignable tenant system roles. Removing a member records
`removed_at` instead of deleting the membership. Owner-requested organization
deletion records `deleted_at`. Internal-administrator suspension records
`suspended_at` and remains separate from organization billing state.
