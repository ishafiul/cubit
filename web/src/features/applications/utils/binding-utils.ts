import type { ResourceBindingType } from '../../../api/model';
import type { ActiveTab } from '../../../App';

export function isServiceAttachedToApp(
  target: string | undefined | null,
  app: { id: string; name: string; subdomain?: string }
): boolean {
  if (!target) return false;
  return target === app.id || target === app.name || (!!app.subdomain && target === app.subdomain);
}

export function getServiceTabForBindingType(type: ResourceBindingType): ActiveTab {
  switch (type) {
    case 'kv_namespace':
      return 'kv';
    case 'd1_database':
      return 'd1';
    case 'r2_bucket':
      return 'r2';
    case 'queue':
      return 'queues';
    case 'workflow':
      return 'workflows';
    case 'durable_object':
      return 'durable-objects';
    case 'container':
      return 'containers';
    case 'assets':
      return 'static-assets';
    case 'service':
      return 'apps';
    default:
      return 'apps';
  }
}

export function getServiceLabelForBindingType(type: ResourceBindingType): string {
  switch (type) {
    case 'kv_namespace':
      return 'KV Namespace';
    case 'd1_database':
      return 'D1 Database';
    case 'r2_bucket':
      return 'R2 Bucket';
    case 'queue':
      return 'Queue Producer';
    case 'workflow':
      return 'Workflow';
    case 'durable_object':
      return 'Durable Object';
    case 'container':
      return 'Container';
    case 'assets':
      return 'Static Assets';
    case 'service':
      return 'Worker RPC';
    default:
      return type;
  }
}

export function getServicePageName(type: ResourceBindingType): string {
  switch (type) {
    case 'kv_namespace':
      return 'KV Namespaces';
    case 'd1_database':
      return 'D1 Databases';
    case 'r2_bucket':
      return 'R2 Buckets';
    case 'queue':
      return 'Queues';
    case 'workflow':
      return 'Workflows';
    case 'durable_object':
      return 'Durable Objects';
    case 'container':
      return 'Containers';
    case 'assets':
      return 'Static Assets';
    case 'service':
      return 'Workers';
    default:
      return 'Service';
  }
}
