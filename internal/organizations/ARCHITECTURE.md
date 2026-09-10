# Organizations Architecture

The organizations domain owns organizations and organization memberships.
The users domain separately owns user identities and their Clerk identity links.

Organization memberships connect users to organizations and carry the
predefined tenant role and access status. Creating an organization and its
active owner membership is one PostgreSQL transaction so neither record can
exist without the other.
