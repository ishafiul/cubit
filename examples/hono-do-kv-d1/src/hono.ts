/**
 * Lightweight, zero-dependency Hono-compatible micro-router for Cloudflare Workers
 * Provides identical API interface: app.get, app.post, app.delete, c.json, c.html, c.req.param, c.req.query
 */

export interface HonoEnv {
  Bindings?: Record<string, any>;
  Variables?: Record<string, any>;
}

export class Context<E extends HonoEnv = HonoEnv> {
  req: {
    raw: Request;
    param: (key: string) => string;
    query: (key: string) => string | undefined;
    header: (key: string) => string | null;
    json: <B = any>() => Promise<B>;
    text: () => Promise<string>;
  };
  env: E['Bindings'] extends Record<string, any> ? E['Bindings'] : any;
  executionCtx: any;
  private params: Record<string, string>;
  private url: URL;

  constructor(request: Request, env: any, executionCtx: any, params: Record<string, string> = {}) {
    this.env = env;
    this.executionCtx = executionCtx;
    this.params = params;
    this.url = new URL(request.url);

    this.req = {
      raw: request,
      param: (key: string) => this.params[key] || '',
      query: (key: string) => this.url.searchParams.get(key) ?? undefined,
      header: (key: string) => request.headers.get(key),
      json: async <B = any>() => (await request.json()) as B,
      text: async () => await request.text(),
    };
  }

  json(data: any, status: number = 200, headers: Record<string, string> = {}): Response {
    return new Response(JSON.stringify(data), {
      status,
      headers: {
        'content-type': 'application/json; charset=utf-8',
        ...headers,
      },
    });
  }

  html(htmlStr: string, status: number = 200, headers: Record<string, string> = {}): Response {
    return new Response(htmlStr, {
      status,
      headers: {
        'content-type': 'text/html; charset=utf-8',
        ...headers,
      },
    });
  }

  text(textStr: string, status: number = 200, headers: Record<string, string> = {}): Response {
    return new Response(textStr, {
      status,
      headers: {
        'content-type': 'text/plain; charset=utf-8',
        ...headers,
      },
    });
  }
}

type Handler<E extends HonoEnv> = (c: Context<E>) => Response | Promise<Response>;

interface Route<E extends HonoEnv> {
  method: string;
  pattern: RegExp;
  paramNames: string[];
  handler: Handler<E>;
}

export class Hono<E extends HonoEnv = HonoEnv> {
  private routes: Route<E>[] = [];

  private register(method: string, path: string, handler: Handler<E>) {
    const paramNames: string[] = [];
    const regexPattern = path
      .replace(/:([a-zA-Z0-9_]+)/g, (_, name) => {
        paramNames.push(name);
        return '([^/]+)';
      })
      .replace(/\//g, '\\/');

    const pattern = new RegExp(`^${regexPattern}\\/?$`);
    this.routes.push({ method, pattern, paramNames, handler });
  }

  get(path: string, handler: Handler<E>): this {
    this.register('GET', path, handler);
    return this;
  }

  post(path: string, handler: Handler<E>): this {
    this.register('POST', path, handler);
    return this;
  }

  put(path: string, handler: Handler<E>): this {
    this.register('PUT', path, handler);
    return this;
  }

  delete(path: string, handler: Handler<E>): this {
    this.register('DELETE', path, handler);
    return this;
  }

  async fetch(request: Request, env: any, ctx: any): Promise<Response> {
    const url = new URL(request.url);
    const pathname = url.pathname;
    const method = request.method.toUpperCase();

    for (const route of this.routes) {
      if (route.method !== method && route.method !== 'ALL') continue;
      const match = pathname.match(route.pattern);
      if (match) {
        const params: Record<string, string> = {};
        route.paramNames.forEach((name, i) => {
          params[name] = match[i + 1];
        });
        const c = new Context<E>(request, env, ctx, params);
        return await route.handler(c);
      }
    }

    return new Response(JSON.stringify({ error: 'Not Found', path: pathname }), {
      status: 404,
      headers: { 'content-type': 'application/json' },
    });
  }
}
