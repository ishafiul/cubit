import { Hono } from './hono';

export interface Env {
  APP_TITLE: string;
  API_VERSION: string;
  ENVIRONMENT: string;
}

interface User {
  id: string;
  name: string;
  email: string;
  role: 'admin' | 'developer' | 'viewer';
  createdAt: string;
}

// In-memory demo data
const users: User[] = [
  { id: 'usr_1', name: 'Alice Chen', email: 'alice@example.com', role: 'admin', createdAt: '2024-01-15T08:00:00Z' },
  { id: 'usr_2', name: 'Bob Smith', email: 'bob@example.com', role: 'developer', createdAt: '2024-02-20T10:30:00Z' },
  { id: 'usr_3', name: 'Clara Oswald', email: 'clara@example.com', role: 'viewer', createdAt: '2024-03-12T14:45:00Z' },
];

const app = new Hono<{ Bindings: Env }>();

// 1. Root Directory & API Index
app.get('/', (c) => {
  return c.json({
    message: 'Welcome to the Cubit Hono REST API with Swagger UI',
    version: c.env.API_VERSION || '1.0.0',
    environment: c.env.ENVIRONMENT || 'production',
    documentation: {
      swaggerUi: '/swagger',
      openApiJson: '/doc',
    },
    endpoints: {
      users: '/api/v1/users',
      health: '/api/v1/health',
    },
  });
});

// 2. OpenAPI 3.0 JSON Specification
app.get('/doc', (c) => {
  const spec = {
    openapi: '3.0.3',
    info: {
      title: c.env.APP_TITLE || 'Cubit Hono REST API',
      version: c.env.API_VERSION || '1.0.0',
      description: 'Production Hono REST API on Cloudflare Workers featuring OpenAPI 3.0 specification and interactive Swagger UI.',
    },
    paths: {
      '/api/v1/health': {
        get: {
          summary: 'Edge Health & Telemetry',
          description: 'Returns worker health status and Cloudflare edge context (colo, country, city, ray-id).',
          responses: {
            '200': {
              description: 'Operational health status',
            },
          },
        },
      },
      '/api/v1/users': {
        get: {
          summary: 'List users',
          description: 'Returns all registered users with optional role filtering.',
          parameters: [
            {
              name: 'role',
              in: 'query',
              required: false,
              schema: { type: 'string', enum: ['admin', 'developer', 'viewer'] },
            },
          ],
          responses: {
            '200': {
              description: 'Array of users',
            },
          },
        },
        post: {
          summary: 'Create user',
          description: 'Registers a new user record.',
          requestBody: {
            required: true,
            content: {
              'application/json': {
                schema: {
                  type: 'object',
                  required: ['name', 'email'],
                  properties: {
                    name: { type: 'string', example: 'Diana Prince' },
                    email: { type: 'string', example: 'diana@example.com' },
                    role: { type: 'string', enum: ['admin', 'developer', 'viewer'], default: 'developer' },
                  },
                },
              },
            },
          },
          responses: {
            '201': { description: 'User successfully created' },
            '400': { description: 'Validation error' },
          },
        },
      },
      '/api/v1/users/{id}': {
        get: {
          summary: 'Get user by ID',
          parameters: [
            { name: 'id', in: 'path', required: true, schema: { type: 'string' } },
          ],
          responses: {
            '200': { description: 'User record' },
            '404': { description: 'User not found' },
          },
        },
        delete: {
          summary: 'Delete user',
          parameters: [
            { name: 'id', in: 'path', required: true, schema: { type: 'string' } },
          ],
          responses: {
            '200': { description: 'User deleted' },
            '404': { description: 'User not found' },
          },
        },
      },
    },
  };

  return c.json(spec);
});

// 3. Interactive Swagger UI
app.get('/swagger', (c) => {
  const html = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Cubit Hono REST API - Swagger UI</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css" />
  <link rel="icon" type="image/png" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/favicon-32x32.png" />
  <style>
    body { margin: 0; background: #fafafa; font-family: sans-serif; }
    .topbar { display: none !important; }
    .swagger-ui .info { margin: 25px 0; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
  <script>
    window.onload = function() {
      window.ui = SwaggerUIBundle({
        url: '/doc',
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIStandalonePreset
        ],
        layout: "BaseLayout"
      });
    };
  </script>
</body>
</html>`;

  return c.html(html);
});

// 4. REST Endpoints
app.get('/api/v1/health', (c) => {
  const cf = (c.req.raw as any).cf || {};
  return c.json({
    status: 'ok',
    uptime: '100%',
    timestamp: new Date().toISOString(),
    edge: {
      colo: cf.colo || 'LOCAL',
      country: cf.country || 'US',
      city: cf.city || 'Localhost',
      asn: cf.asn || 0,
      rayId: c.req.header('CF-Ray') || 'local-ray-id',
    },
  });
});

app.get('/api/v1/users', (c) => {
  const role = c.req.query('role');
  if (role) {
    return c.json(users.filter(u => u.role === role));
  }
  return c.json(users);
});

app.get('/api/v1/users/:id', (c) => {
  const id = c.req.param('id');
  const user = users.find(u => u.id === id);
  if (!user) {
    return c.json({ error: 'User not found' }, 404);
  }
  return c.json(user);
});

app.post('/api/v1/users', async (c) => {
  try {
    const body = await c.req.json<{ name: string; email: string; role?: 'admin' | 'developer' | 'viewer' }>();
    if (!body.name || !body.email) {
      return c.json({ error: 'name and email are required fields' }, 400);
    }

    const newUser: User = {
      id: `usr_${Date.now()}`,
      name: body.name.trim(),
      email: body.email.trim(),
      role: body.role || 'developer',
      createdAt: new Date().toISOString(),
    };

    users.push(newUser);
    return c.json(newUser, 201);
  } catch (err: any) {
    return c.json({ error: 'Invalid JSON body: ' + err.message }, 400);
  }
});

app.delete('/api/v1/users/:id', (c) => {
  const id = c.req.param('id');
  const index = users.findIndex(u => u.id === id);
  if (index === -1) {
    return c.json({ error: 'User not found' }, 404);
  }
  const deleted = users.splice(index, 1)[0];
  return c.json({ message: 'User deleted', user: deleted });
});

export default app;
