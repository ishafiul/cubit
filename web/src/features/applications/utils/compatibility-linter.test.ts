import { describe, it, expect } from 'vitest';
import {
  analyzeCompatibility,
  analyzeApplicationCompatibility,
  analyzeSourceCode,
} from './compatibility-linter';
import type { Application } from '../../../api/model';

describe('compatibility-linter', () => {
  it('identifies compatible worker code and ingress features', () => {
    const code = `
      export default {
        async fetch(request, env, ctx) {
          const clientIp = request.cf?.clientIP;
          ctx.waitUntil(Promise.resolve());
          const cache = caches.default;
          const { 0: client, 1: server } = new WebSocketPair();
          return new Response("OK");
        }
      }
    `;

    const findings = analyzeSourceCode(code);
    expect(findings.some((f) => f.name === 'request.cf Edge Context')).toBe(true);
    expect(findings.some((f) => f.name === 'ctx.waitUntil Lifecycle Hook')).toBe(true);
    expect(findings.some((f) => f.name === 'Cache API (caches.default)')).toBe(true);
    expect(findings.some((f) => f.name === 'WebSocketPair API')).toBe(true);

    const report = analyzeCompatibility(undefined, code);
    expect(report.level).toBe('compatible');
    expect(report.canDeploy).toBe(true);
    expect(report.unsupportedCount).toBe(0);
    expect(report.supportedCount).toBeGreaterThanOrEqual(4);
  });

  it('detects unsupported Cloudflare APIs in code with remediation guidance', () => {
    const code = `
      import { Ai } from '@cloudflare/ai';
      export default {
        async fetch(request, env, ctx) {
          const aiRes = await env.AI.run('@cf/meta/llama-3-8b-instruct', { prompt: "hi" });
          const vec = await env.VECTORIZE.query([0.1, 0.2]);
          const hd = env.HYPERDRIVE;
          const analytics = env.ANALYTICS;
          return new Response("OK");
        }
      }
    `;

    const report = analyzeCompatibility(undefined, code);
    expect(report.level).toBe('incompatible');
    expect(report.canDeploy).toBe(false);
    expect(report.unsupportedCount).toBeGreaterThanOrEqual(4);

    const aiFinding = report.unsupportedFeatures.find((f) => f.name.includes('Workers AI'));
    expect(aiFinding).toBeDefined();
    expect(aiFinding?.remediation).toContain('proxy calls to an external OpenAI/Ollama');

    const vecFinding = report.unsupportedFeatures.find((f) => f.name.includes('Vectorize'));
    expect(vecFinding).toBeDefined();
    expect(vecFinding?.remediation).toContain('pgvector');
  });

  it('inspects parsed wrangler.json config bindings and flags warnings/unsupported items', () => {
    const wranglerConfig = JSON.stringify({
      name: 'multi-service-worker',
      compatibility_date: '2024-09-23',
      kv_namespaces: [{ binding: 'MY_KV', id: 'kv-123' }],
      d1_databases: [{ binding: 'DB', database_id: 'db-456' }],
      r2_buckets: [{ binding: 'BUCKET', bucket_name: 'media-bucket' }],
      durable_objects: {
        bindings: [{ name: 'CHAT_ROOM', class_name: 'ChatRoom' }],
      },
      workflows: [{ binding: 'MY_WORKFLOW', name: 'order-flow', class_name: 'OrderWorkflow' }],
      containers: [{ name: 'RENDERER', image: 'my-renderer:latest', port: 3000 }],
      assets: { directory: './dist' },
      triggers: { crons: ['0 * * * *'] },
      queues: { producers: [{ binding: 'ORDER_QUEUE', queue: 'orders' }] },
      ai: { binding: 'AI' },
    });

    const report = analyzeCompatibility(wranglerConfig);
    expect(report.level).toBe('incompatible'); // due to ai
    expect(report.canDeploy).toBe(false);
    expect(report.supportedFeatures.some((f) => f.name.includes('KV Namespace'))).toBe(true);
    expect(report.supportedFeatures.some((f) => f.name.includes('D1 Database'))).toBe(true);
    expect(report.supportedFeatures.some((f) => f.name.includes('R2 Bucket'))).toBe(true);
    expect(report.supportedFeatures.some((f) => f.name.includes('Durable Object'))).toBe(true);
    expect(report.supportedFeatures.some((f) => f.name.includes('Workflow'))).toBe(true);
    expect(report.supportedFeatures.some((f) => f.name.includes('Container'))).toBe(true);
    expect(report.supportedFeatures.some((f) => f.name.includes('Static Assets'))).toBe(true);
    expect(report.warnings.some((f) => f.name.includes('Queue'))).toBe(true);
    expect(report.unsupportedFeatures.some((f) => f.name.includes('Workers AI'))).toBe(true);
  });

  it('evaluates Application domain models with bindings and inline code', () => {
    const app: Application = {
      id: 'app-1',
      name: 'prod-worker',
      sourceType: 'inline',
      inlineCode: `
        export default {
          async fetch(request, env) {
            return new Response(request.cf.city);
          }
        }
      `,
      bindings: [
        { name: 'KV_STORE', type: 'kv_namespace', resourceId: 'kv-1' },
        { name: 'SQL_DB', type: 'd1_database', resourceId: 'd1-1' },
        { name: 'CONTAINER_APP', type: 'container', resourceId: 'web-svc' },
        { name: 'QUEUE_OUT', type: 'queue', resourceId: 'q-1' },
      ],
      status: 'running',
      createdAt: '2026-01-01T00:00:00Z',
      updatedAt: '2026-01-01T00:00:00Z',
    };

    const report = analyzeApplicationCompatibility(app);
    expect(report.level).toBe('warning'); // queue produces a warning, no unsupported features
    expect(report.canDeploy).toBe(true);
    expect(report.warningCount).toBe(1);
    expect(report.supportedCount).toBeGreaterThanOrEqual(3);
    expect(report.summary).toContain('1 warning');
  });
});
