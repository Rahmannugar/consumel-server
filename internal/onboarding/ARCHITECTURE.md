# Onboarding Architecture

The onboarding domain owns the first-owner setup transaction. It creates the
organization, built-in roles, owner membership, first project, Sandbox, Live,
welcome-email delivery, and outbox event atomically.

The transaction locks the authenticated Consumel user before checking existing
access. Concurrent or repeated submissions therefore return the existing first
project instead of creating duplicate organizations. A project starts without
API keys. API-key creation belongs to the project environment workflow.
