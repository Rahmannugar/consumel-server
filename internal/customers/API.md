## Public integration API

POST /v1/customers
Creates a customer in the project environment authenticated by the API key.

GET /v1/customers
Returns a cursor-paginated customer list from the project environment authenticated by the API key.

GET /v1/customers/{customerId}
Returns one customer from the project environment authenticated by the API key.

PUT /v1/customers/{customerId}
Replaces one customer's optional contact information and supported metadata in the project environment authenticated by the API key.

## Dashboard application API

POST /v1/projects/{projectId}/environments/{environment}/customers
Creates a customer from the signed-in project workspace.

GET /v1/projects/{projectId}/environments/{environment}/customers
Returns a cursor-paginated customer list for the signed-in project workspace.

GET /v1/projects/{projectId}/environments/{environment}/customers/{customerId}
Returns one customer for the signed-in project workspace.

PUT /v1/projects/{projectId}/environments/{environment}/customers/{customerId}
Replaces one customer's optional contact information and supported metadata from the signed-in project workspace.
