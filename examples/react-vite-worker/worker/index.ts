/**
 * Cloudflare Worker Backend + Static Assets Host for React + Vite
 */

export interface Env {
  APP_TITLE?: string;
  ASSETS?: Fetcher;
}

export default {
  async fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
    const url = new URL(request.url);

    // 1. API Endpoints
    if (url.pathname === '/api/health') {
      const cf = (request as any).cf || {};
      return Response.json({
        status: 'healthy',
        message: 'Hello from Cloudflare Worker backing React + Vite!',
        edge: {
          colo: cf.colo || 'LOCAL',
          country: cf.country || 'US',
          rayId: request.headers.get('cf-ray') || 'local-ray',
        },
        timestamp: new Date().toISOString(),
      });
    }

    if (url.pathname === '/api/time') {
      return Response.json({
        epoch: Date.now(),
        iso: new Date().toISOString(),
      });
    }

    // 2. Static Assets via env.ASSETS
    if (env.ASSETS) {
      try {
        const response = await env.ASSETS.fetch(request);
        if (response.status !== 404) {
          return response;
        }
        // SPA Fallback: Serve index.html for client-side routing
        const spaRequest = new Request(new URL('/index.html', request.url), request);
        const spaResponse = await env.ASSETS.fetch(spaRequest);
        if (spaResponse.status !== 404) {
          return spaResponse;
        }
      } catch (_) {
        // pass through to fallback
      }
    }

    // Standalone fallback HTML when assets binding is not attached
    const fallbackHtml = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>React + Vite Worker</title>
    <style>
      body { background: #0f172a; color: #f8fafc; font-family: sans-serif; display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; }
      .card { background: #1e293b; padding: 2rem; border-radius: 12px; border: 1px solid #334155; max-width: 480px; text-align: center; }
      .badge { background: #0284c7; color: white; padding: 4px 10px; border-radius: 999px; font-size: 11px; font-weight: 700; }
      a { color: #38bdf8; text-decoration: none; }
    </style>
  </head>
  <body>
    <div class="card">
      <span class="badge">CLOUDFLARE WORKER</span>
      <h2>React + Vite Static Assets Worker</h2>
      <p>Worker backend active. Test API endpoints at <a href="/api/health">/api/health</a> or <a href="/api/time">/api/time</a>.</p>
    </div>
  </body>
</html>`;

    return new Response(fallbackHtml, {
      headers: { 'content-type': 'text/html; charset=utf-8' },
    });
  },
};
