import { describe, it, expect, beforeEach } from 'vitest';
import { renderHook } from '@testing-library/react';
import {
  useDeploymentTrackerStore,
  useDeploymentTrackerActions,
  type LogEntry,
} from './useDeploymentTrackerStore';

describe('Given the global useDeploymentTrackerStore', () => {
  beforeEach(() => {
    useDeploymentTrackerStore.getState().actions.resetDeploymentTracker();
  });

  describe('When updating deployment states', () => {
    it('Then sets active deployment id and deploying flag', () => {
      const { setActiveDeploymentId, setIsDeploying } =
        useDeploymentTrackerStore.getState().actions;

      setIsDeploying('app-123');
      expect(useDeploymentTrackerStore.getState().isDeploying).toBe('app-123');

      setActiveDeploymentId('dep-abc');
      expect(useDeploymentTrackerStore.getState().activeDeploymentId).toBe('dep-abc');
      expect(useDeploymentTrackerStore.getState().deployStatus).toBe('building');
    });

    it('Then appends logs and deduplicates identical log messages', () => {
      const { appendLog } = useDeploymentTrackerStore.getState().actions;

      const log1: LogEntry = {
        timestamp: '2026-09-25T12:00:00Z',
        step: 'bundle',
        message: 'Bundling worker script',
        level: 'info',
      };

      appendLog(log1);
      expect(useDeploymentTrackerStore.getState().logs.length).toBe(1);

      // Duplicated step + message is discarded
      appendLog(log1);
      expect(useDeploymentTrackerStore.getState().logs.length).toBe(1);

      const log2: LogEntry = {
        timestamp: '2026-09-25T12:00:01Z',
        step: 'celld',
        message: 'Uploading to Garage S3',
        level: 'info',
      };
      appendLog(log2);
      expect(useDeploymentTrackerStore.getState().logs.length).toBe(2);
    });

    it('Then resetDeploymentTracker clears all deployment state and logs', () => {
      const { setActiveDeploymentId, setIsDeploying, appendLog, resetDeploymentTracker } =
        useDeploymentTrackerStore.getState().actions;

      setIsDeploying('app-1');
      setActiveDeploymentId('dep-1');
      appendLog({
        timestamp: '2026-09-25T12:00:00Z',
        step: 'build',
        message: 'msg',
        level: 'info',
      });

      resetDeploymentTracker();

      const state = useDeploymentTrackerStore.getState();
      expect(state.activeDeploymentId).toBeNull();
      expect(state.isDeploying).toBeNull();
      expect(state.deployStatus).toBe('');
      expect(state.logs).toEqual([]);
    });
  });

  describe('When calling useDeploymentTrackerActions', () => {
    it('Then actions reference is stable across state changes', () => {
      const { result, rerender } = renderHook(() => useDeploymentTrackerActions());
      const initialActions = result.current;

      useDeploymentTrackerStore.getState().actions.setDeployStatus('active');
      rerender();

      expect(result.current).toBe(initialActions);
    });
  });
});
