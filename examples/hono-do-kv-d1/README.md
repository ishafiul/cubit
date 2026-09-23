# Hono with Durable Objects, KV, and D1 Example

This example demonstrates building a full-stack, multi-paradigm stateful application using **Hono** on Cloudflare Workers combining:
1. **Durable Objects (`COUNTER_DO`)**: Strongly-consistent distributed actor instances with persistent state.
2. **KV Namespace (`KV_CACHE`)**: High-throughput distributed key-value storage.
3. **D1 Database (`DB`)**: Cloudflare's serverless SQLite relational SQL database.

## Configuration (`wrangler.jsonc`)
```jsonc
{
  "name": "hono-do-kv-d1",
  "main": "src/index.ts",
  "compatibility_date": "2024-09-23",
  "compatibility_flags": ["nodejs_compat"],
  "kv_namespaces": [
    {
      "binding": "KV_CACHE",
      "id": "app_kv_cache"
    }
  ],
  "d1_databases": [
    {
      "binding": "DB",
      "database_name": "app_db",
      "database_id": "app_db_id"
    }
  ],
  "durable_objects": {
    "bindings": [
      {
        "name": "COUNTER_DO",
        "class_name": "CounterDO"
      }
    ]
  }
}
```

## API Routes

### System
- `GET /`: Health and binding configuration dashboard.

### KV Cache
- `GET /api/kv/:key`: Read value from KV.
- `POST /api/kv/:key`: Write value to KV (`{ "value": "my-data" }`).
- `DELETE /api/kv/:key`: Remove key.

### D1 SQLite Database
- `GET /api/d1/notes`: Query recent notes from D1.
- `POST /api/d1/notes`: Insert a new note (`{ "title": "Deploy on Cubit", "content": "Running D1 SQLite on bare metal" }`).
- `DELETE /api/d1/notes/:id`: Delete a note.

### Durable Objects
- `GET /api/do/counter/:name`: Get current count of named Durable Object instance.
- `POST /api/do/counter/:name/increment`: Atomically increment the state of named instance.

## Deploying to Cubit
1. In the Cubit dashboard, open **New Worker** -> **GitHub App**.
2. Select your repository.
3. In **Worker Root Directory**, choose `examples/hono-do-kv-d1`.
4. Click **Create & Deploy**.
