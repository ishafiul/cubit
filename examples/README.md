# Cubit Cloudflare Worker Examples

This directory contains production-ready, standalone Cloudflare Worker example projects designed for deployment on Cubit or Cloudflare.

Each example is self-contained within its own directory and includes a `wrangler.jsonc` (or `wrangler.json`), complete TypeScript source code, and full configuration for bindings and dependencies.

---

## Example Projects Directory

| Example | Directory | Primary Features & Cloudflare Bindings |
| :--- | :--- | :--- |
| **Worker RPC & Service Bindings** | [`examples/worker-rpc`](./worker-rpc) | `services` bindings, Worker-to-Worker RPC calls, WorkerEntrypoint |
| **Hono REST API + Swagger** | [`examples/hono-rest-swagger`](./hono-rest-swagger) | Hono web framework, OpenAPI 3.0 specification, Interactive Swagger UI |
| **vNext (Next.js Edge)** | [`examples/vnext`](./vnext) | Next.js Edge Runtime, Server Components, Route Handlers, OpenNext |
| **React + Vite Worker** | [`examples/react-vite-worker`](./react-vite-worker) | React 18 frontend, Vite build tool, Workers Static Assets (`assets`) |
| **Hono + DO + KV + D1** | [`examples/hono-do-kv-d1`](./hono-do-kv-d1) | Durable Objects, Cloudflare KV namespace, Cloudflare D1 (SQLite) |

---

## Deploying to Cubit via GitHub

Thanks to Cubit's monorepo subfolder discovery, you can deploy any of these examples directly from your GitHub repository:

1. Open the Cubit Dashboard in your browser (`http://localhost:8000`).
2. Click **New Worker** (or **Deploy Worker**).
3. Select the **GitHub App** source tab.
4. Select your repository (e.g. `cubit`).
5. In the **Worker Root Directory / Folder** dropdown, select one of the examples:
   - `examples/worker-rpc` ⚡
   - `examples/hono-rest-swagger` ⚡
   - `examples/vnext` ⚡
   - `examples/react-vite-worker` ⚡
   - `examples/hono-do-kv-d1` ⚡
6. Click **Create & Deploy**. Cubit will automatically pull the branch, navigate to the selected folder, parse its `wrangler.jsonc`, bundle the entrypoint, and activate your worker!
