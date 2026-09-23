# React + Vite Cloudflare Worker Example

This example demonstrates building a modern **React** frontend using **Vite** and serving it from a **Cloudflare Worker** using **Workers Static Assets** (`env.ASSETS`).

## Features
- **Vite + React 18**: Instant HMR and optimized production bundles.
- **Workers Static Assets**: Zero-overhead static file delivery directly at the edge.
- **SPA Fallback**: Unmatched paths fall back to `index.html` for client-side routing.
- **Edge Backend APIs**: Handles API endpoints like `/api/health` and `/api/time`.

## Configuration (`wrangler.jsonc`)
```jsonc
{
  "name": "react-vite-worker",
  "main": "worker/index.ts",
  "compatibility_date": "2024-09-23",
  "assets": {
    "directory": "./dist",
    "binding": "ASSETS"
  }
}
```

## Running Locally
```bash
# 1. Start Vite dev server for frontend
npm run dev

# 2. Build for production
npm run build

# 3. Preview locally with Wrangler
npx wrangler dev
```

## Deploying to Cubit
1. In the Cubit dashboard, open **New Worker** -> **GitHub App**.
2. Select your repository.
3. In **Worker Root Directory**, choose `examples/react-vite-worker`.
4. Click **Create & Deploy**.
