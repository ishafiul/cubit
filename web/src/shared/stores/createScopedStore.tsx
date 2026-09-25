import { createContext, useContext, useRef, type FC, type PropsWithChildren } from 'react';
import { createStore, StoreApi } from 'zustand/vanilla';
import { useStore as useZustandStore } from 'zustand';

export type StoreStateWithActions<TState, TActions> = TState & { actions: TActions };

export type ScopedStoreApi<TState, TActions> = StoreApi<StoreStateWithActions<TState, TActions>>;

export type StoreInitializer<TState, TActions, TProps> = (
  props: TProps,
  set: ScopedStoreApi<TState, TActions>['setState'],
  get: ScopedStoreApi<TState, TActions>['getState']
) => StoreStateWithActions<TState, TActions>;

export interface ScopedStore<TState, TActions, TProps> {
  Provider: FC<PropsWithChildren<TProps extends void ? {} : TProps>>;
  useStore: () => ScopedStoreApi<TState, TActions>;
  useSelector: <TResult>(selector: (state: TState) => TResult) => TResult;
  useActions: () => TActions;
}

export function createScopedStore<
  TState extends Record<string, any>,
  TActions extends Record<string, any>,
  TProps = void
>(
  initializer: StoreInitializer<TState, TActions, TProps>,
  storeName = 'ScopedStore'
): ScopedStore<TState, TActions, TProps> {
  const Context = createContext<ScopedStoreApi<TState, TActions> | null>(null);
  Context.displayName = `${storeName}Context`;

  function Provider({ children, ...props }: PropsWithChildren<any>) {
    const storeRef = useRef<ScopedStoreApi<TState, TActions> | null>(null);

    if (!storeRef.current) {
      storeRef.current = createStore<StoreStateWithActions<TState, TActions>>((set, get) =>
        initializer(props as TProps, set, get)
      );
    }

    return <Context.Provider value={storeRef.current}>{children}</Context.Provider>;
  }

  function useStore(): ScopedStoreApi<TState, TActions> {
    const store = useContext(Context);
    if (!store) {
      throw new Error(`${storeName} hooks must be used within its Provider`);
    }
    return store;
  }

  function useSelector<TResult>(selector: (state: TState) => TResult): TResult {
    const store = useStore();
    return useZustandStore(store, (state) => selector(state as TState));
  }

  function useActions(): TActions {
    const store = useStore();
    return useZustandStore(store, (state) => state.actions);
  }

  return {
    Provider,
    useStore,
    useSelector,
    useActions,
  };
}
