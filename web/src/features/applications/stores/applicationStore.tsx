import { createScopedStore } from '../../../shared/stores/createScopedStore';

export type DetailTab = 'overview' | 'code' | 'builds' | 'triggers' | 'bindings' | 'settings';

export type MetricsInterval = '1h' | '24h' | '7d' | '30d';

export interface ApplicationState {
  appId: string;
  activeTab: DetailTab;
  draftCode: string;
  savedCode: string;
  isDirty: boolean;
  activeMetricsInterval: MetricsInterval;
  secretVisibility: Record<string, boolean>;
  logFilter: string;
  logLevelFilter: string;
}

export interface ApplicationActions {
  setActiveTab: (tab: DetailTab) => void;
  setDraftCode: (code: string) => void;
  markCodeSaved: (code: string) => void;
  resetDraftCode: () => void;
  setActiveMetricsInterval: (interval: MetricsInterval) => void;
  toggleSecretVisibility: (key: string) => void;
  setLogFilter: (filter: string) => void;
  setLogLevelFilter: (level: string) => void;
}

export interface ApplicationStoreProps {
  appId: string;
  initialTab?: DetailTab;
  initialCode?: string;
}

export const {
  Provider: ApplicationStoreProvider,
  useStore: useApplicationStore,
  useSelector: useApplicationSelector,
  useActions: useApplicationActions,
} = createScopedStore<ApplicationState, ApplicationActions, ApplicationStoreProps>(
  (props, set) => {
    const initialCode = props.initialCode ?? '';

    const actions: ApplicationActions = {
      setActiveTab: (activeTab) => set({ activeTab }),

      setDraftCode: (draftCode) =>
        set((state) => ({
          draftCode,
          isDirty: draftCode !== state.savedCode,
        })),

      markCodeSaved: (savedCode) =>
        set({
          savedCode,
          draftCode: savedCode,
          isDirty: false,
        }),

      resetDraftCode: () =>
        set((state) => ({
          draftCode: state.savedCode,
          isDirty: false,
        })),

      setActiveMetricsInterval: (activeMetricsInterval) =>
        set({ activeMetricsInterval }),

      toggleSecretVisibility: (key) =>
        set((state) => ({
          secretVisibility: {
            ...state.secretVisibility,
            [key]: !state.secretVisibility[key],
          },
        })),

      setLogFilter: (logFilter) => set({ logFilter }),

      setLogLevelFilter: (logLevelFilter) => set({ logLevelFilter }),
    };

    return {
      appId: props.appId,
      activeTab: props.initialTab ?? 'overview',
      draftCode: initialCode,
      savedCode: initialCode,
      isDirty: false,
      activeMetricsInterval: '24h',
      secretVisibility: {},
      logFilter: '',
      logLevelFilter: 'all',
      actions,
    };
  },
  'ApplicationStore'
);

// Shortcut hooks for high-frequency consumption with granular zero/minimal re-renders
export function useActiveTab(): DetailTab {
  return useApplicationSelector((state) => state.activeTab);
}

export function useIsDirty(): boolean {
  return useApplicationSelector((state) => state.isDirty);
}

export function useDraftCode(): string {
  return useApplicationSelector((state) => state.draftCode);
}

export function useSecretVisibility(key: string): boolean {
  return useApplicationSelector((state) => Boolean(state.secretVisibility[key]));
}

export function useActiveMetricsInterval(): MetricsInterval {
  return useApplicationSelector((state) => state.activeMetricsInterval);
}

export function useLogFilter(): string {
  return useApplicationSelector((state) => state.logFilter);
}

export function useLogLevelFilter(): string {
  return useApplicationSelector((state) => state.logLevelFilter);
}
