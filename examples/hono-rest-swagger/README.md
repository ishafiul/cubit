# Hono REST API with Swagger UI Example

This example demonstrates a fast, lightweight REST API built with [Hono](https://hono.dev/) on Cloudflare Workers, featuring embedded OpenAPI 3.0 documentation and an interactive **Swagger UI**.

## Features
- **Hono Web Framework**: Sub-millisecond routing and JSON serialization.
- **Embedded Swagger UI**: Interactive API documentation explorer available at `/swagger`.
- **OpenAPI 3.0 Specification**: Machine-readable specification served at `/doc`.
- **CRUD Operations**: User management endpoints (`GET`, `POST`, `DELETE`).
- **Edge Telemetry**: Returns Cloudflare edge context (`request.cf`) at `/api/v1/health`.

## Endpoints
- `GET /`: API entrypoint and directory links.
- `GET /swagger`: Interactive Swagger UI.
- `GET /doc`: OpenAPI 3.0 JSON specification.
- `GET /api/v1/health`: Cloudflare Edge context (colo, country, city, ray-id).
- `GET /api/v1/users`: List users (supports `?role=admin`).
- `GET /api/v1/users/:id`: Get user details.
- `POST /api/v1/users`: Create user (`{ "name": "...", "email": "..." }`).
- `DELETE /api/v1/users/:id`: Remove user.

## Deploying to Cubit
1. In the Cubit dashboard, click **New Worker** -> **GitHub App**.
2. Select your repository.
3. In **Worker Root Directory**, choose `examples/hono-rest-swagger`.
4. Click **Create & Deploy**.
5. Once deployed, open the test URL and visit `/swagger` to test the API!
