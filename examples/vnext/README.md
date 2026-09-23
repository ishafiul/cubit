# vNext - Next.js on Cloudflare Workers Example

This example demonstrates deploying a full-stack **Next.js** application to Cloudflare Workers using the **Edge Runtime** and **OpenNext**.

## Features
- **Next.js Edge Runtime**: Ultra-fast SSR with zero cold start execution.
- **Route Handlers**: API routes (`/api/hello`, `/api/posts`) running directly in worker isolates.
- **Static Assets (`assets`)**: Static assets (`/_next/static`, `/public`) served directly from local storage/CDN.
- **OpenNext Config**: Ready for `@opennextjs/cloudflare` build pipelines.

## Routes
- `GET /`: Next.js Server Components SSR landing page with live edge metadata.
- `GET /api/hello`: Next.js Route Handler returning edge location details.
- `GET /api/posts`: Next.js dynamic Route Handler.

## Deploying to Cubit
1. In the Cubit dashboard, open **New Worker** -> **GitHub App**.
2. Select your repository.
3. In **Worker Root Directory**, choose `examples/vnext`.
4. Click **Create & Deploy**.
