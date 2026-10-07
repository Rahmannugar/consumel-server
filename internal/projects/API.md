GET /v1/projects
Returns the signed-in user's active projects with their Sandbox and Live environments.

GET /v1/projects/portfolio
Returns environment-scoped request outcomes and per-project summaries across the signed-in organization.

POST /v1/projects
Creates a project with isolated Sandbox and Live environments in the signed-in user's organization.

GET /v1/projects/{projectId}/environments/{environment}/api-key
Returns safe metadata for the selected environment's active API key without returning its secret.

POST /v1/projects/{projectId}/environments/{environment}/api-key
Creates the selected environment's first API key and returns its plaintext only in this response.

POST /v1/projects/{projectId}/environments/{environment}/api-key/replace
Atomically revokes the selected environment's active API key and returns its replacement plaintext only in this response.

DELETE /v1/projects/{projectId}/environments/{environment}/api-key
Revokes the selected environment's active API key.

POST /v1/projects/{projectId}/environments/{environment}/activate
Activates the selected environment explicitly and returns its current activation state.
