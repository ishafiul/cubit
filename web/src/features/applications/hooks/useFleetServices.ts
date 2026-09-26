import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { Application } from '../../../api/model';

export interface AttachedCronTrigger {
  id: string;
  name: string;
  cronExpression?: string;
  cron?: string;
  targetAppId?: string;
  target_app_id?: string;
  status?: string;
  lastRunAt?: string;
  last_run_at?: string;
  nextRunAt?: string;
  next_run_at?: string;
}

export interface AttachedQueue {
  id: string;
  name: string;
  consumerAppId?: string;
  consumer_app_id?: string;
  maxRetries?: number;
  max_retries?: number;
  maxBatchSize?: number;
  max_batch_size?: number;
  backlogCount?: number;
  messagesProcessed?: number;
}

export interface AttachedDurableObject {
  id: string;
  name: string;
  appId?: string;
  app_id?: string;
  facets?: any[];
}

export interface AttachedWorkflow {
  id: string;
  name: string;
  targetAppId?: string;
  target_app_id?: string;
  steps?: string[];
}

export function isServiceAttachedToApp(
  target: string | undefined | null,
  app: { id: string; name: string; subdomain?: string }
): boolean {
  if (!target) return false;
  return target === app.id || target === app.name || (!!app.subdomain && target === app.subdomain);
}

export function getServiceTabForBindingType(type: string): string {
  switch (type) {
    case 'kv':
    case 'kv_namespace':
      return 'kv';
    case 'd1':
    case 'd1_database':
      return 'd1';
    case 'r2':
    case 'r2_bucket':
      return 'r2';
    case 'queue':
      return 'queues';
    case 'workflow':
      return 'workflows';
    case 'durable_object':
      return 'durable-objects';
    case 'service':
      return 'apps';
    default:
      return 'apps';
  }
}

export function useFleetServices(app: Application) {
  const queryClient = useQueryClient();
  const queryKey = ['fleet-services', app.id];

  const query = useQuery({
    queryKey,
    queryFn: async () => {
      const [cronRes, queuesRes, doRes, wfRes, kvRes, d1Res, r2Res] = await Promise.all([
        fetch('/api/v1/cron').catch(() => null),
        fetch('/api/v1/queues').catch(() => null),
        fetch('/api/v1/durable-objects').catch(() => null),
        fetch('/api/v1/workflows').catch(() => null),
        fetch('/api/v1/kv/namespaces').catch(() => null),
        fetch('/api/v1/d1/databases').catch(() => null),
        fetch('/api/v1/r2/buckets').catch(() => null),
      ]);

      const appIdMatches = (target?: string | null) => isServiceAttachedToApp(target, app);

      let attachedCron: AttachedCronTrigger[] = [];
      let attachedQueues: AttachedQueue[] = [];
      let attachedDO: AttachedDurableObject[] = [];
      let attachedWorkflows: AttachedWorkflow[] = [];

      let fleetKV: any[] = [];
      let fleetD1: any[] = [];
      let fleetR2: any[] = [];
      let fleetQueues: any[] = [];
      let fleetWorkflows: any[] = [];

      if (cronRes?.ok) {
        const data = await cronRes.json().catch(() => []);
        const list = Array.isArray(data) ? data : [];
        attachedCron = list.filter((c: any) =>
          appIdMatches(c.targetAppId || c.target_app_id)
        );
      }

      if (queuesRes?.ok) {
        const data = await queuesRes.json().catch(() => []);
        const list = Array.isArray(data) ? data : [];
        fleetQueues = list;
        attachedQueues = list.filter((q: any) =>
          appIdMatches(q.consumerAppId || q.consumer_app_id)
        );
      }

      if (doRes?.ok) {
        const data = await doRes.json().catch(() => []);
        const list = Array.isArray(data) ? data : [];
        attachedDO = list.filter((d: any) =>
          appIdMatches(d.appId || d.app_id)
        );
      }

      if (wfRes?.ok) {
        const data = await wfRes.json().catch(() => []);
        const list = Array.isArray(data) ? data : [];
        fleetWorkflows = list;
        attachedWorkflows = list.filter((w: any) =>
          appIdMatches(w.targetAppId || w.target_app_id)
        );
      }

      if (kvRes?.ok) {
        const data = await kvRes.json().catch(() => []);
        fleetKV = Array.isArray(data) ? data : [];
      }

      if (d1Res?.ok) {
        const data = await d1Res.json().catch(() => []);
        fleetD1 = Array.isArray(data) ? data : [];
      }

      if (r2Res?.ok) {
        const data = await r2Res.json().catch(() => []);
        fleetR2 = Array.isArray(data) ? data : [];
      }

      return {
        attachedCron,
        attachedQueues,
        attachedDO,
        attachedWorkflows,
        fleetKV,
        fleetD1,
        fleetR2,
        fleetQueues,
        fleetWorkflows,
      };
    },
    staleTime: 10_000,
  });

  const runCronMutation = useMutation({
    mutationFn: async (cronId: string) => {
      const res = await fetch(`/api/v1/cron/${cronId}/run`, { method: 'POST' });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error || 'Failed to trigger cron');
      }
      return res.json().catch(() => ({}));
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey });
    },
  });

  const createCronMutation = useMutation({
    mutationFn: async (data: { name: string; cron: string; targetAppId: string }) => {
      const res = await fetch('/api/v1/cron', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: data.name,
          cron: data.cron,
          target_app_id: data.targetAppId,
        }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error || 'Failed to create cron trigger');
      }
      return res.json().catch(() => ({}));
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey });
    },
  });

  const data = query.data || {
    attachedCron: [],
    attachedQueues: [],
    attachedDO: [],
    attachedWorkflows: [],
    fleetKV: [],
    fleetD1: [],
    fleetR2: [],
    fleetQueues: [],
    fleetWorkflows: [],
  };

  return {
    ...data,
    isLoading: query.isLoading,
    refetch: query.refetch,
    runCronNow: runCronMutation.mutateAsync,
    isRunningCron: runCronMutation.isPending,
    createCron: createCronMutation.mutateAsync,
    isCreatingCron: createCronMutation.isPending,
  };
}
