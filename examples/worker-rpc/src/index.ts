/**
 * Cloudflare Worker RPC & Service Bindings Example
 *
 * Demonstrates:
 * 1. Consuming another worker using Worker-to-Worker Service bindings (env.AUTH_SERVICE, env.CALCULATOR_SERVICE).
 * 2. Exporting an RPC class extending WorkerEntrypoint for zero-overhead direct method calls.
 * 3. Fallback mock handlers for standalone local execution.
 */

export interface Env {
  AUTH_SERVICE?: Fetcher;
  CALCULATOR_SERVICE?: Fetcher;
  ENVIRONMENT: string;
  SERVICE_NAME: string;
}

// 1. WorkerEntrypoint exposing direct RPC methods to caller workers
export class MathServiceRpc {
  add(a: number, b: number): number {
    return a + b;
  }

  multiply(a: number, b: number): number {
    return a * b;
  }

  computeTax(subtotal: number, taxRate: number = 0.08): { subtotal: number; tax: number; total: number } {
    const tax = Math.round(subtotal * taxRate * 100) / 100;
    return {
      subtotal,
      tax,
      total: subtotal + tax,
    };
  }
}

// 2. Gateway Fetch Handler
export default {
  async fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
    const url = new URL(request.url);

    // Health check endpoint
    if (url.pathname === '/health') {
      return Response.json({
        status: 'healthy',
        service: env.SERVICE_NAME || 'worker-rpc',
        timestamp: new Date().toISOString(),
      });
    }

    // Call Target Worker via Service Binding RPC / Fetch
    if (url.pathname === '/api/calculate' && request.method === 'POST') {
      try {
        const body = (await request.json()) as { a?: number; b?: number; op?: string };
        const a = Number(body.a ?? 10);
        const b = Number(body.b ?? 5);
        const op = body.op || 'add';

        // If target service binding is available, call it via Service binding
        if (env.CALCULATOR_SERVICE) {
          const svcResp = await env.CALCULATOR_SERVICE.fetch('http://calculator-service/calculate', {
            method: 'POST',
            headers: { 'content-type': 'application/json' },
            body: JSON.stringify({ a, b, op }),
          });
          const result = await svcResp.json();
          return Response.json({
            source: 'service_binding',
            data: result,
          });
        }

        // Fallback: Local RPC evaluation
        const math = new MathServiceRpc();
        let result: number;
        switch (op) {
          case 'multiply':
            result = math.multiply(a, b);
            break;
          case 'add':
          default:
            result = math.add(a, b);
            break;
        }

        return Response.json({
          source: 'local_rpc_entrypoint',
          operation: op,
          a,
          b,
          result,
        });
      } catch (err: any) {
        return Response.json({ error: err.message }, { status: 400 });
      }
    }

    // Auth Service Verification via Service Binding
    if (url.pathname === '/api/auth/verify' && request.method === 'POST') {
      const authHeader = request.headers.get('Authorization') || '';
      const token = authHeader.replace(/^Bearer\s+/i, '');

      if (env.AUTH_SERVICE) {
        const authResp = await env.AUTH_SERVICE.fetch('http://auth-service/verify', {
          method: 'POST',
          headers: { 'content-type': 'application/json' },
          body: JSON.stringify({ token }),
        });
        return authResp;
      }

      // Standalone simulation
      const isValid = token.length > 5;
      return Response.json({
        source: 'standalone_mock',
        valid: isValid,
        user: isValid ? { id: 'usr_123', email: 'developer@example.com' } : null,
      });
    }

    // Default Gateway landing
    return Response.json({
      service: env.SERVICE_NAME || 'worker-rpc',
      environment: env.ENVIRONMENT || 'production',
      features: [
        'Cloudflare Workers RPC',
        'Worker-to-Worker Service Bindings',
        'Direct WorkerEntrypoint invocation',
      ],
      endpoints: [
        { path: '/health', method: 'GET', description: 'Worker health check' },
        { path: '/api/calculate', method: 'POST', description: 'Execute math RPC via service binding' },
        { path: '/api/auth/verify', method: 'POST', description: 'Validate authorization token via auth service' },
      ],
      boundServices: {
        AUTH_SERVICE: Boolean(env.AUTH_SERVICE),
        CALCULATOR_SERVICE: Boolean(env.CALCULATOR_SERVICE),
      },
    });
  },
};
