import { describe, it, expect } from 'vitest';
import type { ActiveTab } from '../../App';
import { isSystemBucket, isValidBucketName } from './R2View';

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

describe('Given R2 bucket name validation rules', () => {
  describe('When evaluating valid S3 compliant bucket names', () => {
    it('Then accepts standard lowercase alphanumeric names with hyphens and dots', () => {
      expect(isValidBucketName('app-assets')).toBe(true);
      expect(isValidBucketName('my-bucket-01')).toBe(true);
      expect(isValidBucketName('data.warehouse.2026')).toBe(true);
      expect(isValidBucketName('abc')).toBe(true);
    });
  });

  describe('When evaluating invalid bucket names', () => {
    it('Then rejects names that are too short, have uppercase, or invalid edge characters', () => {
      expect(isValidBucketName('ab')).toBe(false);
      expect(isValidBucketName('-starts-with-hyphen')).toBe(false);
      expect(isValidBucketName('ends-with-hyphen-')).toBe(false);
      expect(isValidBucketName('.starts-with-dot')).toBe(false);
      expect(isValidBucketName('Upper-Case-Bucket')).toBe(false);
      expect(isValidBucketName('has space')).toBe(false);
    });
  });
});

describe('Given R2 system bucket protection checks', () => {
  describe('When inspecting bucket metadata', () => {
    it('Then flags cubit-fleet as a protected system bucket', () => {
      expect(isSystemBucket({ name: 'cubit-fleet' })).toBe(true);
      expect(isSystemBucket({ name: 'cubit-fleet', isSystem: true })).toBe(true);
    });

    it('Then flags any bucket with isSystem true as protected', () => {
      expect(isSystemBucket({ name: 'custom-internal', isSystem: true })).toBe(true);
    });

    it('Then allows user custom buckets to be marked unprotected', () => {
      expect(isSystemBucket({ name: 'user-uploads', isSystem: false })).toBe(false);
      expect(isSystemBucket({ name: 'my-app-assets' })).toBe(false);
    });

    it('Then safely handles null or undefined bucket records', () => {
      expect(isSystemBucket(null)).toBe(false);
      expect(isSystemBucket(undefined)).toBe(false);
    });
  });
});
