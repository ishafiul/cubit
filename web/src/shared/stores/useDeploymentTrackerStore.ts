import { create } from 'zustand';

export interface LogEntry {
  timestamp: string;
  step: string;
  message: string;
  level: string;
}

export interface DeploymentTrackerState {
  activeDeploymentId: string | null;
  deployStatus: string;
  isDeploying: string | null;
  logs: LogEntry[];
}

export interface DeploymentTrackerActions {
  setActiveDeploymentId: (id: string | null) => void;
  setDeployStatus: (status: string) => void;
  setIsDeploying: (appId: string | null) => void;
  setLogs: (logs: LogEntry[]) => void;
  appendLog: (entry: LogEntry) => void;
  clearLogs: () => void;
  resetDeploymentTracker: () => void;
}

export type DeploymentTrackerStore = DeploymentTrackerState & {
  actions: DeploymentTrackerActions;
};

const defaultDeploymentTrackerState: DeploymentTrackerState = {
  activeDeploymentId: null,
  deployStatus: '',
  isDeploying: null,
  logs: [],
};

export const useDeploymentTrackerStore = create<DeploymentTrackerStore>((set) => {
  const actions: DeploymentTrackerActions = {
    setActiveDeploymentId: (activeDeploymentId) =>
      set({
        activeDeploymentId,
        deployStatus: activeDeploymentId ? 'building' : '',
        logs: activeDeploymentId ? [] : [],
      }),

    setDeployStatus: (deployStatus) => set({ deployStatus }),

    setIsDeploying: (isDeploying) => set({ isDeploying }),

    setLogs: (logs) => set({ logs }),

    appendLog: (entry) =>
      set((state) => {
        const exists = state.logs.some(
          (p) => p.step === entry.step && p.message === entry.message
        );
        return exists ? state : { logs: [...state.logs, entry] };
      }),

    clearLogs: () => set({ logs: [] }),

    resetDeploymentTracker: () => set({ ...defaultDeploymentTrackerState }),
  };

  return {
    ...defaultDeploymentTrackerState,
    actions,
  };
});

export function useDeploymentTrackerActions(): DeploymentTrackerActions {
  return useDeploymentTrackerStore((state) => state.actions);
}

export function useActiveDeploymentId(): string | null {
  return useDeploymentTrackerStore((state) => state.activeDeploymentId);
}

export function useDeployStatus(): string {
  return useDeploymentTrackerStore((state) => state.deployStatus);
}

export function useIsDeploying(): string | null {
  return useDeploymentTrackerStore((state) => state.isDeploying);
}

export function useDeploymentLogs(): LogEntry[] {
  return useDeploymentTrackerStore((state) => state.logs);
}
