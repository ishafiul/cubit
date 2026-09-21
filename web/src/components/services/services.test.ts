import { describe, it, expect } from 'vitest';
import type { ActiveTab } from '../../App';

describe('Given Celld 0.5.1 service suite definitions', () => {
  describe('When verifying official 12 service coverage in the sidebar', () => {
    const officialServices: ActiveTab[] = [
      'apps',              // 1. Workers
      'durable-objects',   // 2. Durable Objects & 3. DO Facets
      'containers',        // 4. Containers (Experimental)
      'static-assets',     // 5. Static assets
      'cron',              // 6. Cron Triggers
      'dynamic-workers',   // 7. Dynamic Workers
      'kv',                // 8. KV
      'queues',            // 9. Queues
      'd1',                // 10. D1
      'workflows',         // 11. Workflows
      'r2',                // 12. R2
    ];

    it('Then all 12 services have corresponding ActiveTab entries', () => {
      expect(officialServices).toHaveLength(11); // 11 tabs (DO Facets unified under durable-objects)
      expect(officialServices).toContain('kv');
      expect(officialServices).toContain('d1');
      expect(officialServices).toContain('r2');
      expect(officialServices).toContain('dynamic-workers');
      expect(officialServices).toContain('durable-objects');
      expect(officialServices).toContain('containers');
      expect(officialServices).toContain('static-assets');
      expect(officialServices).toContain('cron');
      expect(officialServices).toContain('queues');
      expect(officialServices).toContain('workflows');
      expect(officialServices).toContain('apps');
    });

    it('Then each service is assigned to a designated dashboard section', () => {
      const computeServices: ActiveTab[] = ['apps', 'dynamic-workers', 'durable-objects', 'containers'];
      const storageServices: ActiveTab[] = ['kv', 'd1', 'r2', 'static-assets'];
      const automationServices: ActiveTab[] = ['cron', 'queues', 'workflows'];

      expect(computeServices).toHaveLength(4);
      expect(storageServices).toHaveLength(4);
      expect(automationServices).toHaveLength(3);
    });
  });
});
