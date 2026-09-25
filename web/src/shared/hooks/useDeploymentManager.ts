import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import {
  useDeploymentTrackerActions,
  useActiveDeploymentId,
  useDeployStatus,
  useIsDeploying,
  useDeploymentLogs,
  LogEntry,
} from '../stores/useDeploymentTrackerStore';
import {
  useDeployApplication,
} from '../../api/generated/deployments/deployments';
import { getListApplicationsQueryKey } from '../../api/generated/applications/applications';
import { useToastActions } from '../stores/useToastStore';

export function useDeploymentManager() {
  const queryClient = useQueryClient();
  const { addToast } = useToastActions();
  const deployMutation = useDeployApplication();

  const activeDeploymentId = useActiveDeploymentId();
  const deployStatus = useDeployStatus();
  const isDeploying = useIsDeploying();
  const logs = useDeploymentLogs();
  const {
    setActiveDeploymentId,
    setDeployStatus,
    setIsDeploying,
    setLogs,
    appendLog,
  } = useDeploymentTrackerActions();

  // SSE Stream Effect for active deployment
  useEffect(() => {
    if (!activeDeploymentId) return;

    setLogs([]);
    setDeployStatus('building');

    const origin = typeof window !== 'undefined' && window.location?.origin ? window.location.origin : 'http://localhost';

    fetch(`${origin}/api/v1/deployments/${activeDeploymentId}/logs`)
      .then((r) => r.json())
      .then((data: unknown) => {
        if (Array.isArray(data) && data.length > 0) {
          const formatted: LogEntry[] = data.map((entry: Record<string, string>) => ({
            timestamp: entry.timestamp || entry.Timestamp || new Date().toISOString(),
            step: entry.step || entry.Step || 'info',
            message: entry.message || entry.Message || '',
            level: entry.level || entry.Level || 'info',
          }));
          setLogs(formatted);
        }
      })
      .catch((err) => console.error('Failed fetching initial logs', err));

    fetch(`${origin}/api/v1/deployments/${activeDeploymentId}`)
      .then((r) => r.json())
      .then((data: Record<string, unknown>) => {
        if (data && typeof data.status === 'string') {
          setDeployStatus(data.status);
        }
      })
      .catch((err) => console.error('Failed fetching deployment info', err));

    if (typeof EventSource === 'undefined') return;

    const es = new EventSource(`/api/v1/deployments/${activeDeploymentId}/logs/stream`);

    es.onmessage = (event) => {
      try {
        const entry = JSON.parse(event.data);
        const normalized: LogEntry = {
          timestamp: entry.timestamp || entry.Timestamp || new Date().toISOString(),
          step: entry.step || entry.Step || 'info',
          message: entry.message || entry.Message || '',
          level: entry.level || entry.Level || 'info',
        };
        appendLog(normalized);
      } catch (e) {
        console.error('Failed parsing log entry', e);
      }
    };

    es.addEventListener('complete', (event: MessageEvent) => {
      es.close();
      if (event.data) {
        try {
          const parsed = JSON.parse(event.data);
          if (parsed.status) setDeployStatus(parsed.status);
        } catch (_) {
          setDeployStatus('active');
        }
      } else {
        setDeployStatus('active');
      }
      queryClient.invalidateQueries({ queryKey: getListApplicationsQueryKey() });
      addToast({
        title: 'Deployment Completed',
        description: `Deployment ${activeDeploymentId.slice(0, 8)} finished successfully.`,
        variant: 'success',
      });
    });

    es.onerror = () => {
      es.close();
    };

    return () => {
      es.close();
    };
  }, [activeDeploymentId, appendLog, setDeployStatus, setLogs, queryClient, addToast]);

  const deployApp = async (appId: string) => {
    setIsDeploying(appId);
    try {
      const res = await deployMutation.mutateAsync({
        id: appId,
        data: { commitHash: 'HEAD' },
      });
      if (res && res.id) {
        setActiveDeploymentId(res.id);
        await queryClient.invalidateQueries({ queryKey: getListApplicationsQueryKey() });
        addToast({
          title: 'Deployment Triggered',
          description: `Build started for worker: deployment ${res.id.slice(0, 8)}.`,
          variant: 'info',
        });
      }
    } catch (err: unknown) {
      const errorMsg =
        (err as { response?: { data?: { error?: string } }; message?: string })?.response?.data
          ?.error ||
        (err as { message?: string })?.message ||
        'Failed to trigger deployment.';
      addToast({
        title: 'Deployment Failed',
        description: errorMsg,
        variant: 'error',
      });
      throw err;
    } finally {
      setIsDeploying(null);
    }
  };

  return {
    deployApp,
    isDeploying,
    activeDeploymentId,
    setActiveDeploymentId,
    deployStatus,
    logs,
  };
}
