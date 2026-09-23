# Worker RPC & Service Bindings Example

This example demonstrates Cloudflare Workers **Service Bindings** and **Workers RPC** (zero-cost in-isolate inter-worker communication).

## Features
- **Service Bindings**: Declaratively calls peer workers (`env.AUTH_SERVICE`, `env.CALCULATOR_SERVICE`) without going over the public internet.
- **WorkerEntrypoint RPC**: Implements direct method invocations (`MathServiceRpc`).
- **Gateway Pattern**: Aggregates peer microservices behind a unified API gateway.

## Configuration (`wrangler.jsonc`)
```jsonc
{
  "name": "worker-rpc",
  "main": "src/index.ts",
  "compatibility_date": "2024-09-23",
  "compatibility_flags": ["nodejs_compat"],
  "services": [
    {
      "binding": "AUTH_SERVICE",
      "service": "auth-service"
    },
    {
      "binding": "CALCULATOR_SERVICE",
      "service": "calculator-service"
    }
  ]
}
```

## Endpoints
- `GET /`: Gateway overview and bound services state.
- `GET /health`: Liveness probe.
- `POST /api/calculate`: Dispatches calculation to the `CALCULATOR_SERVICE` service binding.
  ```json
  { "a": 20, "b": 15, "op": "multiply" }
  ```
- `POST /api/auth/verify`: Validates a token using the `AUTH_SERVICE` service binding.

## Deploying to Cubit
1. In the Cubit dashboard, open **New Worker** -> **GitHub App**.
2. Select your repository.
3. In **Worker Root Directory**, choose `examples/worker-rpc`.
4. Click **Create & Deploy**.
