## Public integration API

POST /v1/meters
Creates a meter configuration in the project environment authenticated by the API key.

GET /v1/meters
Returns a cursor-paginated, searchable meter list from the project environment authenticated by the API key.

GET /v1/meters/{meterKey}
Returns one meter from the project environment authenticated by the API key.

PUT /v1/meters/{meterKey}
Replaces the editable name and optional description of one project-level meter definition authenticated by the API key.

## Dashboard application API

POST /v1/projects/{projectId}/environments/{environment}/meters
Creates a meter configuration from the signed-in project workspace.

GET /v1/projects/{projectId}/environments/{environment}/meters
Returns a cursor-paginated, searchable meter list for the signed-in project workspace.

GET /v1/projects/{projectId}/environments/{environment}/meters/{meterKey}
Returns one meter from the signed-in project workspace.

PUT /v1/projects/{projectId}/environments/{environment}/meters/{meterKey}
Replaces the editable name and optional description of one project-level meter definition from the signed-in project workspace.
