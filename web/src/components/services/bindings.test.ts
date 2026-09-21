import { describe, it, expect } from 'vitest';
import {
  getServiceTabForBindingType,
  getServiceLabelForBindingType,
  getServicePageName,
  isServiceAttachedToApp,
} from '../ApplicationDetailPage';

describe('Given resource binding type to dashboard service mappings', () => {
  describe('When mapping binding types to corresponding sidebar tabs', () => {
    it('Then kv_namespace maps to kv tab', () => {
      expect(getServiceTabForBindingType('kv_namespace')).toBe('kv');
    });

    it('Then d1_database maps to d1 tab', () => {
      expect(getServiceTabForBindingType('d1_database')).toBe('d1');
    });

    it('Then r2_bucket maps to r2 tab', () => {
      expect(getServiceTabForBindingType('r2_bucket')).toBe('r2');
    });

    it('Then queue maps to queues tab', () => {
      expect(getServiceTabForBindingType('queue')).toBe('queues');
    });

    it('Then workflow maps to workflows tab', () => {
      expect(getServiceTabForBindingType('workflow')).toBe('workflows');
    });
  });

  describe('When formatting display labels and target page names', () => {
    it('Then human-readable service labels are returned for each type', () => {
      expect(getServiceLabelForBindingType('kv_namespace')).toBe('KV Namespace');
      expect(getServiceLabelForBindingType('d1_database')).toBe('D1 Database');
      expect(getServiceLabelForBindingType('r2_bucket')).toBe('R2 Bucket');
      expect(getServiceLabelForBindingType('queue')).toBe('Queue Producer');
      expect(getServiceLabelForBindingType('workflow')).toBe('Workflow');
    });

    it('Then service control panel destination titles are accurately formatted', () => {
      expect(getServicePageName('kv_namespace')).toBe('KV Namespaces');
      expect(getServicePageName('d1_database')).toBe('D1 Databases');
      expect(getServicePageName('r2_bucket')).toBe('R2 Buckets');
      expect(getServicePageName('queue')).toBe('Queues');
      expect(getServicePageName('workflow')).toBe('Workflows');
    });
  });
});

describe('Given a worker application and fleet service targets', () => {
  const app = {
    id: 'f5754de1-42da-9ac3-a5e0-176c44d33b48',
    name: 'hello-world-worker',
    subdomain: 'hello-world-worker',
  };

  describe('When matching reverse fleet service references', () => {
    it('Then matches when target equals app ID', () => {
      expect(isServiceAttachedToApp('f5754de1-42da-9ac3-a5e0-176c44d33b48', app)).toBe(true);
    });

    it('Then matches when target equals app name', () => {
      expect(isServiceAttachedToApp('hello-world-worker', app)).toBe(true);
    });

    it('Then does not match when target belongs to another worker or is empty', () => {
      expect(isServiceAttachedToApp('other-app-id', app)).toBe(false);
      expect(isServiceAttachedToApp('', app)).toBe(false);
      expect(isServiceAttachedToApp(undefined, app)).toBe(false);
      expect(isServiceAttachedToApp(null, app)).toBe(false);
    });
  });
});
