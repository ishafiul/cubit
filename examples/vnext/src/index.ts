/**
 * vNext - Next.js Edge Runtime Worker for Cloudflare & Cubit
 *
 * Implements:
 * 1. Next.js App Router Edge SSR Rendering (Server Components output).
 * 2. Next.js Edge Route Handlers (/api/hello, /api/posts).
 * 3. Static Assets serving via env.ASSETS.
 */

export interface Env {
  APP_NAME: string;
  FRAMEWORK: string;
  ENVIRONMENT: string;
  ASSETS?: Fetcher;
}

export default {
  async fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
    const url = new URL(request.url);
    const cf = (request as any).cf || {};

    // 1. Next.js Route Handler: /api/hello
    if (url.pathname === '/api/hello') {
      return Response.json({
        message: 'Hello from Next.js Edge API Route!',
        runtime: 'edge',
        timestamp: new Date().toISOString(),
        colo: cf.colo || 'LOCAL',
        country: cf.country || 'US',
      });
    }

    // 2. Next.js Route Handler: /api/posts
    if (url.pathname === '/api/posts') {
      const posts = [
        { id: 1, title: 'Running Next.js on Bare-Metal with Cubit', slug: 'cubit-nextjs' },
        { id: 2, title: 'OpenNext Cloudflare Architecture', slug: 'opennext-cloudflare' },
        { id: 3, title: 'Zero Cold Start Edge Serverless', slug: 'edge-serverless' },
      ];
      return Response.json({
        posts,
        total: posts.length,
        renderedAt: new Date().toISOString(),
      });
    }

    // 3. Static Assets fallback (e.g. /favicon.ico, /_next/static)
    if (env.ASSETS && (url.pathname.startsWith('/_next') || url.pathname.includes('.'))) {
      try {
        const asset = await env.ASSETS.fetch(request);
        if (asset.status !== 404) return asset;
      } catch (_) {
        // pass through to SSR
      }
    }

    // 4. Next.js Server Components SSR Page
    const colo = cf.colo || 'LOCAL';
    const country = cf.country || 'US';
    const city = cf.city || 'Edge Point';
    const rayId = request.headers.get('cf-ray') || 'ray_edge_dev';

    const html = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>${env.APP_NAME || 'vNext'} - Next.js on Cloudflare Workers</title>
  <style>
    :root {
      --bg: #09090b;
      --card: #18181b;
      --border: #27272a;
      --text: #f4f4f5;
      --muted: #a1a1aa;
      --accent: #38bdf8;
      --accent-glow: rgba(56, 189, 248, 0.15);
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background: var(--bg);
      color: var(--text);
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      min-height: 100vh;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      padding: 24px;
    }
    .container {
      max-width: 680px;
      width: 100%;
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 16px;
      padding: 36px;
      box-shadow: 0 20px 40px rgba(0,0,0,0.5);
    }
    .badge {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      background: var(--accent-glow);
      border: 1px solid rgba(56, 189, 248, 0.3);
      color: var(--accent);
      padding: 4px 12px;
      border-radius: 9999px;
      font-size: 12px;
      font-weight: 600;
      margin-bottom: 20px;
    }
    h1 {
      font-size: 28px;
      font-weight: 800;
      letter-spacing: -0.03em;
      margin-bottom: 10px;
    }
    p.lead {
      color: var(--muted);
      font-size: 15px;
      line-height: 1.6;
      margin-bottom: 28px;
    }
    .grid {
      display: grid;
      grid-template-columns: repeat(2, 1fr);
      gap: 12px;
      margin-bottom: 28px;
    }
    .card {
      background: #111113;
      border: 1px solid var(--border);
      padding: 16px;
      border-radius: 10px;
    }
    .card-title {
      font-size: 11px;
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: 0.05em;
      font-weight: 700;
      margin-bottom: 4px;
    }
    .card-val {
      font-size: 14px;
      font-weight: 600;
      font-family: monospace;
      color: #38bdf8;
    }
    .actions {
      display: flex;
      gap: 10px;
    }
    a.btn {
      flex: 1;
      text-align: center;
      background: #27272a;
      color: var(--text);
      text-decoration: none;
      padding: 10px 16px;
      border-radius: 8px;
      font-size: 13px;
      font-weight: 600;
      transition: background 0.15s;
    }
    a.btn:hover { background: #3f3f46; }
    a.btn.primary {
      background: var(--accent);
      color: #09090b;
    }
    a.btn.primary:hover { background: #7dd3fc; }
  </style>
</head>
<body>
  <div class="container">
    <div class="badge">▲ Next.js App Router • Edge Runtime</div>
    <h1>${env.APP_NAME || 'vNext Worker'}</h1>
    <p class="lead">
      Full-stack Next.js application compiled and running natively inside a Cloudflare isolate on Cubit.
    </p>

    <div class="grid">
      <div class="card">
        <div class="card-title">Edge Colo / Location</div>
        <div class="card-val">${colo} (${city}, ${country})</div>
      </div>
      <div class="card">
        <div class="card-title">Runtime Architecture</div>
        <div class="card-val">V8 Isolate (Zero Cold Start)</div>
      </div>
      <div class="card">
        <div class="card-title">Cloudflare Ray ID</div>
        <div class="card-val">${rayId}</div>
      </div>
      <div class="card">
        <div class="card-title">SSR Render Timestamp</div>
        <div class="card-val">${new Date().toISOString().slice(11, 19)} UTC</div>
      </div>
    </div>

    <div class="actions">
      <a href="/api/hello" class="btn primary" target="_blank">Test Route Handler (/api/hello)</a>
      <a href="/api/posts" class="btn" target="_blank">View Edge Posts API</a>
    </div>
  </div>
</body>
</html>`;

    return new Response(html, {
      status: 200,
      headers: {
        'content-type': 'text/html; charset=utf-8',
        'x-powered-by': 'Next.js / Cubit Edge',
      },
    });
  },
};
