# Organizations Architecture

The organizations domain owns organizations and organization memberships.
The users domain separately owns tenant-user records and their stable Authlier
subject links.

Organization memberships connect users to organizations and carry one
organization role plus an access status. Organizations have built-in Admin and
Developer roles and may define custom roles. Creating an organization, its
built-in roles, and its active owner membership is one PostgreSQL transaction.

Ownership is recorded on the organization through `owner_user_id`, separately
from the role assigned to the owner's membership. Removing a member records
`removed_at` instead of deleting the membership. Owner-requested organization
deletion records `deleted_at`. Internal-administrator suspension records
`suspended_at` and remains separate from organization billing state.
