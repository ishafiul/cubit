import type { Application } from '../../../api/model';

export type CompatibilityLevel = 'compatible' | 'warning' | 'incompatible';
export type FeatureStatus = 'supported' | 'warning' | 'unsupported';
export type FeatureCategory = 'config' | 'binding' | 'source_import' | 'api_usage';

export interface FeatureFinding {
  name: string;
  category: FeatureCategory;
  status: FeatureStatus;
  file?: string;
  line?: number;
  details: string;
  remediation?: string;
}

export interface CompatibilityReport {
  level: CompatibilityLevel;
  canDeploy: boolean;
  totalIssues: number;
  unsupportedCount: number;
  warningCount: number;
  supportedCount: number;
  supportedFeatures: FeatureFinding[];
  unsupportedFeatures: FeatureFinding[];
  warnings: FeatureFinding[];
  summary: string;
}

interface PatternRule {
  regex: RegExp;
  name: string;
  category: FeatureCategory;
  status: FeatureStatus;
  details: string;
  remediation?: string;
}

const sourceRules: PatternRule[] = [
  // Unsupported APIs
  {
    regex: /(?:\bimport\s+.*?["']@cloudflare\/ai["']|\benv\s*\.\s*AI\b|\bAi\s*\.\s*run\b)/,
    name: 'Cloudflare Workers AI (env.AI)',
    category: 'api_usage',
    status: 'unsupported',
    details: 'Workers AI model inference (@cf/*) is not natively embedded in celld.',
    remediation:
      'Cubit runs V8 isolates with private SQLite cells. To run LLM inference, proxy calls to an external OpenAI/Ollama compatible endpoint or self-hosted LLM server via fetch().',
  },
  {
    regex: /(?:\bimport\s+.*?["']@cloudflare\/vectorize["']|\benv\s*\.\s*VECTORIZE\b)/,
    name: 'Cloudflare Vectorize (env.VECTORIZE)',
    category: 'api_usage',
    status: 'unsupported',
    details: 'Vectorize vector database embeddings are not embedded in celld.',
    remediation:
      'Vectorize embeddings are not supported in celld isolates. Connect to PostgreSQL with pgvector, Qdrant, or Chroma via HTTP fetch().',
  },
  {
    regex: /\benv\s*\.\s*HYPERDRIVE\b/,
    name: 'Cloudflare Hyperdrive (env.HYPERDRIVE)',
    category: 'api_usage',
    status: 'unsupported',
    details: 'Hyperdrive connection pooling is a Cloudflare-managed feature.',
    remediation:
      'Hyperdrive connection pooling is a Cloudflare-managed feature. Connect directly to Postgres or MySQL using standard client connection pooling or a TCP proxy.',
  },
  {
    regex: /(?:\bimport\s+.*?["']@cloudflare\/puppeteer["']|\benv\s*\.\s*BROWSER\b)/,
    name: 'Cloudflare Browser Rendering (@cloudflare/puppeteer)',
    category: 'source_import',
    status: 'unsupported',
    details: 'Cloudflare Browser Rendering uses external managed Chromium clusters.',
    remediation:
      'Headless browser instances require external container execution. Launch a dedicated Chromium container and control it via Chrome DevTools Protocol (CDP) over WebSockets.',
  },
  {
    regex: /\benv\s*\.\s*ANALYTICS\b/,
    name: 'Cloudflare Analytics Engine (env.ANALYTICS)',
    category: 'api_usage',
    status: 'unsupported',
    details: 'Analytics Engine datasets are proprietary to Cloudflare edge networks.',
    remediation:
      'Analytics Engine is proprietary to Cloudflare. Use SQLite inside a Durable Object, ClickHouse, or a Prometheus metrics endpoint.',
  },

  // Supported APIs
  {
    regex: /\breq(?:uest)?\s*\.\s*cf\b/,
    name: 'request.cf Edge Context',
    category: 'api_usage',
    status: 'supported',
    details: 'Cloudflare edge metadata (country, city, clientIP, colo, rayID) synthesized by Cubit ingress.',
  },
  {
    regex: /\bcaches\s*\.\s*default\b/,
    name: 'Cache API (caches.default)',
    category: 'api_usage',
    status: 'supported',
    details: 'Edge cache simulation supported in isolate runtime.',
  },
  {
    regex: /\bnew\s+WebSocketPair\b/,
    name: 'WebSocketPair API',
    category: 'api_usage',
    status: 'supported',
    details: 'Bidirectional WebSockets fully supported in celld and isolate runtime.',
  },
  {
    regex: /\bctx\s*\.\s*waitUntil\b/,
    name: 'ctx.waitUntil Lifecycle Hook',
    category: 'api_usage',
    status: 'supported',
    details: 'Asynchronous background task lifecycle fully supported.',
  },
];

export function analyzeSourceCode(code: string, fileName = 'src/index.ts'): FeatureFinding[] {
  const findings: FeatureFinding[] = [];
  if (!code) return findings;

  const lines = code.split('\n');
  const seenFeatures = new Set<string>();

  for (let i = 0; i < lines.length; i++) {
    const lineText = lines[i];
    for (const rule of sourceRules) {
      if (rule.regex.test(lineText)) {
        const key = `${rule.name}:${rule.status}`;
        if (seenFeatures.has(key)) continue;
        seenFeatures.add(key);

        findings.push({
          name: rule.name,
          category: rule.category,
          status: rule.status,
          file: fileName,
          line: i + 1,
          details: rule.details,
          remediation: rule.remediation,
        });
      }
    }
  }

  return findings;
}

export function analyzeCompatibility(rawConfigOrCode?: string, sourceCode?: string): CompatibilityReport {
  const supported: FeatureFinding[] = [];
  const unsupported: FeatureFinding[] = [];
  const warnings: FeatureFinding[] = [];

  let parsedConfig: any = null;
  let codeToScan = sourceCode || '';

  if (rawConfigOrCode) {
    const trimmed = rawConfigOrCode.trim();
    if (trimmed.startsWith('{')) {
      try {
        parsedConfig = JSON.parse(trimmed);
      } catch (_) {
        // Not valid JSON; might be source code or TOML
      }
    }

    if (!parsedConfig) {
      // Check if it's TOML or code
      if (trimmed.includes('export default') || trimmed.includes('addEventListener') || trimmed.includes('import ')) {
        codeToScan = codeToScan ? `${codeToScan}\n${rawConfigOrCode}` : rawConfigOrCode;
      }
    }
  }

  // Scan parsed JSON config if available
  if (parsedConfig && typeof parsedConfig === 'object') {
    // KV Namespaces
    if (Array.isArray(parsedConfig.kv_namespaces)) {
      for (const kv of parsedConfig.kv_namespaces) {
        supported.push({
          name: `KV Namespace (${kv.binding || 'KV'})`,
          category: 'binding',
          status: 'supported',
          details: `Backed by SQLite key-value store (id: ${kv.id || 'auto'})`,
        });
      }
    }

    // D1 Databases
    if (Array.isArray(parsedConfig.d1_databases)) {
      for (const db of parsedConfig.d1_databases) {
        supported.push({
          name: `D1 Database (${db.binding || 'DB'})`,
          category: 'binding',
          status: 'supported',
          details: `Backed by private SQLite database (id: ${db.database_id || db.id || 'auto'})`,
        });
      }
    }

    // R2 Buckets
    if (Array.isArray(parsedConfig.r2_buckets)) {
      for (const r2 of parsedConfig.r2_buckets) {
        supported.push({
          name: `R2 Bucket (${r2.binding || 'BUCKET'})`,
          category: 'binding',
          status: 'supported',
          details: `Backed by Garage S3 distributed bucket (${r2.bucket_name || 'bucket'})`,
        });
      }
    }

    // Durable Objects
    if (parsedConfig.durable_objects?.bindings && Array.isArray(parsedConfig.durable_objects.bindings)) {
      for (const doBinding of parsedConfig.durable_objects.bindings) {
        supported.push({
          name: `Durable Object (${doBinding.name})`,
          category: 'binding',
          status: 'supported',
          details: `Backed by celld private SQLite cell for class ${doBinding.class_name}`,
        });
      }
    }

    // Workflows
    if (Array.isArray(parsedConfig.workflows)) {
      for (const wf of parsedConfig.workflows) {
        supported.push({
          name: `Workflow (${wf.binding || wf.name})`,
          category: 'binding',
          status: 'supported',
          details: `Durable execution engine workflow ${wf.name || wf.binding} (class: ${wf.class_name || 'Workflow'})`,
        });
      }
    }

    // Containers
    if (Array.isArray(parsedConfig.containers)) {
      for (const ct of parsedConfig.containers) {
        supported.push({
          name: `Container (${ct.name})`,
          category: 'binding',
          status: 'supported',
          details: `Docker container workload ${ct.name} (${ct.image}:${ct.port || 8080})`,
        });
      }
    }

    // Services
    if (Array.isArray(parsedConfig.services)) {
      for (const s of parsedConfig.services) {
        supported.push({
          name: `Service Binding (${s.binding})`,
          category: 'binding',
          status: 'supported',
          details: `Worker-to-worker invocation target: ${s.service}`,
        });
      }
    }

    // Assets
    if (parsedConfig.assets && (parsedConfig.assets.directory || typeof parsedConfig.assets === 'string')) {
      const dir = typeof parsedConfig.assets === 'string' ? parsedConfig.assets : parsedConfig.assets.directory;
      supported.push({
        name: 'Static Assets',
        category: 'config',
        status: 'supported',
        details: `Directory "${dir}" served directly from Garage S3 / HTTP cache`,
      });
    }

    // Triggers / Crons
    if (parsedConfig.triggers?.crons && Array.isArray(parsedConfig.triggers.crons)) {
      supported.push({
        name: 'Cron Triggers',
        category: 'config',
        status: 'supported',
        details: `${parsedConfig.triggers.crons.length} scheduled trigger expression(s)`,
      });
    }

    // Queues (Warning)
    if (parsedConfig.queues) {
      warnings.push({
        name: 'Queue Producers / Consumers',
        category: 'config',
        status: 'warning',
        details: 'Queues require external message broker or DO emulation.',
        remediation: 'Queue messages can be dispatched using SQLite Durable Objects or external Redis/RabbitMQ message queues.',
      });
    }

    // Unsupported Cloudflare features in config
    if (parsedConfig.ai) {
      unsupported.push({
        name: 'Workers AI Binding (config.ai)',
        category: 'config',
        status: 'unsupported',
        details: 'Cloudflare AI binding configured in wrangler config.',
        remediation: 'Workers AI inference is not natively embedded in celld. Use fetch() to an OpenAI/Ollama compatible endpoint.',
      });
    }

    if (parsedConfig.vectorize && (Array.isArray(parsedConfig.vectorize) ? parsedConfig.vectorize.length > 0 : true)) {
      unsupported.push({
        name: 'Vectorize Bindings (config.vectorize)',
        category: 'config',
        status: 'unsupported',
        details: 'Cloudflare Vectorize database configured in wrangler config.',
        remediation: 'Vectorize embeddings are not supported in celld isolates. Connect to PostgreSQL with pgvector, Qdrant, or Chroma via HTTP fetch().',
      });
    }

    if (parsedConfig.hyperdrive && (Array.isArray(parsedConfig.hyperdrive) ? parsedConfig.hyperdrive.length > 0 : true)) {
      unsupported.push({
        name: 'Hyperdrive Bindings (config.hyperdrive)',
        category: 'config',
        status: 'unsupported',
        details: 'Cloudflare Hyperdrive database connection pooling configured.',
        remediation: 'Connect directly to Postgres or MySQL using standard client connection pooling or a TCP proxy.',
      });
    }

    if (parsedConfig.browser) {
      unsupported.push({
        name: 'Browser Rendering Binding (config.browser)',
        category: 'config',
        status: 'unsupported',
        details: 'Headless Browser Rendering configured.',
        remediation: 'Launch a dedicated Chromium container and control it via Chrome DevTools Protocol (CDP) over WebSockets.',
      });
    }

    if (parsedConfig.analytics_engine_datasets) {
      unsupported.push({
        name: 'Analytics Engine Datasets',
        category: 'config',
        status: 'unsupported',
        details: 'Cloudflare Analytics Engine configured.',
        remediation: 'Use SQLite inside a Durable Object, ClickHouse, or a Prometheus metrics endpoint.',
      });
    }
  }

  // Scan source code
  if (codeToScan) {
    const codeFindings = analyzeSourceCode(codeToScan);
    for (const finding of codeFindings) {
      if (finding.status === 'unsupported') {
        unsupported.push(finding);
      } else if (finding.status === 'warning') {
        warnings.push(finding);
      } else if (finding.status === 'supported') {
        supported.push(finding);
      }
    }
  }

  const supportedCount = supported.length;
  const unsupportedCount = unsupported.length;
  const warningCount = warnings.length;
  const totalIssues = unsupportedCount + warningCount;

  let level: CompatibilityLevel = 'compatible';
  let canDeploy = true;
  let summary = '';

  if (unsupportedCount > 0) {
    level = 'incompatible';
    canDeploy = false;
    summary = `Project has ${unsupportedCount} incompatible Cloudflare feature(s) requiring remediation before deploying to celld.`;
  } else if (warningCount > 0) {
    level = 'warning';
    canDeploy = true;
    summary = `Project is deployable with ${warningCount} warning(s). Review remediation steps for best performance.`;
  } else {
    level = 'compatible';
    canDeploy = true;
    summary = `Project is fully compatible with celld (${supportedCount} supported feature(s) detected).`;
  }

  return {
    level,
    canDeploy,
    totalIssues,
    unsupportedCount,
    warningCount,
    supportedCount,
    supportedFeatures: supported,
    unsupportedFeatures: unsupported,
    warnings,
    summary,
  };
}

export function analyzeApplicationCompatibility(app: Application): CompatibilityReport {
  const supported: FeatureFinding[] = [];
  const unsupported: FeatureFinding[] = [];
  const warnings: FeatureFinding[] = [];

  // Inspect application bindings
  if (app.bindings && Array.isArray(app.bindings)) {
    for (const b of app.bindings) {
      switch (b.type) {
        case 'kv_namespace':
          supported.push({
            name: `KV Namespace (${b.name})`,
            category: 'binding',
            status: 'supported',
            details: `Target resource: ${b.resourceId || 'internal KV'}`,
          });
          break;
        case 'd1_database':
          supported.push({
            name: `D1 Database (${b.name})`,
            category: 'binding',
            status: 'supported',
            details: `Target SQLite database: ${b.resourceId || 'internal D1'}`,
          });
          break;
        case 'r2_bucket':
          supported.push({
            name: `R2 Bucket (${b.name})`,
            category: 'binding',
            status: 'supported',
            details: `Target Garage S3 bucket: ${b.resourceId || 'internal R2'}`,
          });
          break;
        case 'durable_object':
          supported.push({
            name: `Durable Object (${b.name})`,
            category: 'binding',
            status: 'supported',
            details: `Celld private SQLite cell for class: ${b.resourceId || 'DurableObject'}`,
          });
          break;
        case 'container':
          supported.push({
            name: `Container Workload (${b.name})`,
            category: 'binding',
            status: 'supported',
            details: `Target Docker container: ${b.resourceId}`,
          });
          break;
        case 'assets':
          supported.push({
            name: `Static Assets (${b.name})`,
            category: 'binding',
            status: 'supported',
            details: `Asset path: ${b.resourceId || 'public'}`,
          });
          break;
        case 'workflow':
          supported.push({
            name: `Workflow (${b.name})`,
            category: 'binding',
            status: 'supported',
            details: `Workflow execution target: ${b.resourceId}`,
          });
          break;
        case 'service':
          supported.push({
            name: `Service Binding (${b.name})`,
            category: 'binding',
            status: 'supported',
            details: `Target worker: ${b.resourceId}`,
          });
          break;
        case 'queue':
          warnings.push({
            name: `Queue Binding (${b.name})`,
            category: 'binding',
            status: 'warning',
            details: `Queue producer target: ${b.resourceId}`,
            remediation: 'Queues require external message broker or SQLite DO emulation.',
          });
          break;
      }
    }
  }

  // Inspect source code
  if (app.inlineCode) {
    const codeFindings = analyzeSourceCode(app.inlineCode);
    for (const finding of codeFindings) {
      if (finding.status === 'unsupported') {
        unsupported.push(finding);
      } else if (finding.status === 'warning') {
        warnings.push(finding);
      } else if (finding.status === 'supported') {
        supported.push(finding);
      }
    }
  }

  const supportedCount = supported.length;
  const unsupportedCount = unsupported.length;
  const warningCount = warnings.length;
  const totalIssues = unsupportedCount + warningCount;

  let level: CompatibilityLevel = 'compatible';
  let canDeploy = true;
  let summary = '';

  if (unsupportedCount > 0) {
    level = 'incompatible';
    canDeploy = false;
    summary = `Worker has ${unsupportedCount} incompatible Cloudflare feature(s) requiring remediation before running on celld.`;
  } else if (warningCount > 0) {
    level = 'warning';
    canDeploy = true;
    summary = `Worker is deployable with ${warningCount} warning(s). Review remediation recommendations.`;
  } else {
    level = 'compatible';
    canDeploy = true;
    summary = `Worker is fully compatible with celld (${supportedCount} supported feature(s) detected).`;
  }

  return {
    level,
    canDeploy,
    totalIssues,
    unsupportedCount,
    warningCount,
    supportedCount,
    supportedFeatures: supported,
    unsupportedFeatures: unsupported,
    warnings,
    summary,
  };
}
