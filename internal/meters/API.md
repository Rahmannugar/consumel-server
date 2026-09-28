## Public integration API

POST /v1/meters
Creates a meter configuration in the project environment authenticated by the API key.

GET /v1/meters
Returns a cursor-paginated meter list from the project environment authenticated by the API key.

GET /v1/meters/{meterKey}
Returns one meter from the project environment authenticated by the API key.

## Dashboard application API

POST /v1/projects/{projectId}/environments/{environment}/meters
Creates a meter configuration from the signed-in project workspace.

GET /v1/projects/{projectId}/environments/{environment}/meters
Returns a cursor-paginated meter list for the signed-in project workspace.

GET /v1/projects/{projectId}/environments/{environment}/meters/{meterKey}
Returns one meter from the signed-in project workspace.
