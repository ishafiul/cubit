import { Hono } from './hono';
export { CounterDO } from './durable-object';

export interface Env {
  KV_CACHE?: KVNamespace;
  DB?: D1Database;
  COUNTER_DO?: DurableObjectNamespace;
  APP_NAME?: string;
}

const app = new Hono<{ Bindings: Env }>();

// 1. Root Status Dashboard
app.get('/', async (c) => {
  return c.json({
    app: c.env.APP_NAME || 'Hono State Suite',
    bindings: {
      kvCache: Boolean(c.env.KV_CACHE),
      databaseD1: Boolean(c.env.DB),
      durableObject: Boolean(c.env.COUNTER_DO),
    },
    documentation: {
      kv: [
        'GET /api/kv/:key',
        'POST /api/kv/:key (body: { value })',
        'DELETE /api/kv/:key',
      ],
      d1: [
        'GET /api/d1/notes',
        'POST /api/d1/notes (body: { title, content })',
        'DELETE /api/d1/notes/:id',
      ],
      durableObjects: [
        'GET /api/do/counter/:name',
        'POST /api/do/counter/:name/increment',
      ],
    },
  });
});

// 2. Cloudflare KV Endpoints
app.get('/api/kv/:key', async (c) => {
  const key = c.req.param('key');
  if (!c.env.KV_CACHE) {
    return c.json({ key, value: `[simulated-kv-value-for-${key}]`, note: 'KV binding not configured' });
  }
  const value = await c.env.KV_CACHE.get(key);
  if (value === null) {
    return c.json({ error: 'Key not found' }, 404);
  }
  return c.json({ key, value });
});

app.post('/api/kv/:key', async (c) => {
  const key = c.req.param('key');
  const body = await c.req.json<{ value: any; expirationTtl?: number }>();
  if (!c.env.KV_CACHE) {
    return c.json({ success: true, key, value: body.value, note: 'KV binding simulated' });
  }
  const valStr = typeof body.value === 'string' ? body.value : JSON.stringify(body.value);
  await c.env.KV_CACHE.put(key, valStr, { expirationTtl: body.expirationTtl });
  return c.json({ success: true, key, value: body.value });
});

app.delete('/api/kv/:key', async (c) => {
  const key = c.req.param('key');
  if (c.env.KV_CACHE) {
    await c.env.KV_CACHE.delete(key);
  }
  return c.json({ success: true, message: `Key ${key} deleted` });
});

// 3. Cloudflare D1 Database Endpoints
app.get('/api/d1/notes', async (c) => {
  if (!c.env.DB) {
    return c.json({
      notes: [
        { id: 'sim_1', title: 'Local D1 Simulation', content: 'D1 binding ready', priority: 1 },
      ],
      source: 'simulated_fallback',
    });
  }
  try {
    const { results } = await c.env.DB.prepare('SELECT * FROM notes ORDER BY created_at DESC LIMIT 50').all();
    return c.json({ notes: results || [] });
  } catch (err: any) {
    return c.json({ error: 'D1 query failed: ' + err.message }, 500);
  }
});

app.post('/api/d1/notes', async (c) => {
  const body = await c.req.json<{ title: string; content: string; priority?: number }>();
  if (!body.title || !body.content) {
    return c.json({ error: 'title and content are required' }, 400);
  }
  const id = `note_${Date.now()}`;
  if (!c.env.DB) {
    return c.json({ success: true, note: { id, ...body }, source: 'simulated_fallback' }, 201);
  }
  try {
    await c.env.DB.prepare(
      'INSERT INTO notes (id, title, content, priority) VALUES (?, ?, ?, ?)'
    ).bind(id, body.title, body.content, body.priority || 1).run();
    return c.json({ success: true, note: { id, ...body } }, 201);
  } catch (err: any) {
    return c.json({ error: 'Failed inserting note: ' + err.message }, 500);
  }
});

app.delete('/api/d1/notes/:id', async (c) => {
  const id = c.req.param('id');
  if (c.env.DB) {
    await c.env.DB.prepare('DELETE FROM notes WHERE id = ?').bind(id).run();
  }
  return c.json({ success: true, deleted: id });
});

// 4. Cloudflare Durable Objects Endpoints
app.get('/api/do/counter/:name', async (c) => {
  const name = c.req.param('name');
  if (!c.env.COUNTER_DO) {
    return c.json({ name, count: 1, source: 'standalone_mock' });
  }
  const id = c.env.COUNTER_DO.idFromName(name);
  const stub = c.env.COUNTER_DO.get(id);
  const resp = await stub.fetch('http://durable-object/state');
  return resp;
});

app.post('/api/do/counter/:name/increment', async (c) => {
  const name = c.req.param('name');
  if (!c.env.COUNTER_DO) {
    return c.json({ name, action: 'incremented', count: 42, source: 'standalone_mock' });
  }
  const id = c.env.COUNTER_DO.idFromName(name);
  const stub = c.env.COUNTER_DO.get(id);
  const resp = await stub.fetch('http://durable-object/increment', { method: 'POST' });
  return resp;
});

export default app;
