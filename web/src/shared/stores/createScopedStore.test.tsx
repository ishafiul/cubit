import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, act, cleanup } from '@testing-library/react';
import { createScopedStore } from './createScopedStore';

afterEach(() => {
  cleanup();
});

interface CounterState {
  count: number;
  text: string;
}

interface CounterActions {
  increment: () => void;
  setText: (text: string) => void;
}

interface CounterProps {
  initialCount?: number;
}

const {
  Provider: CounterProvider,
  useSelector: useCounterSelector,
  useActions: useCounterActions,
  useStore: useCounterStore,
} = createScopedStore<CounterState, CounterActions, CounterProps>(
  (props, set) => ({
    count: props.initialCount ?? 0,
    text: 'initial',
    actions: {
      increment: () => set((state) => ({ count: state.count + 1 })),
      setText: (text) => set(() => ({ text })),
    },
  })
);

describe('Given a scoped store created with createScopedStore', () => {
  describe('When rendered inside its Provider', () => {
    it('Then selectors read the initial state configured by props', () => {
      function CounterView() {
        const count = useCounterSelector((s) => s.count);
        return <div data-testid="count">{count}</div>;
      }

      render(
        <CounterProvider initialCount={10}>
          <CounterView />
        </CounterProvider>
      );

      expect(screen.getByTestId('count').textContent).toBe('10');
    });

    it('Then actions update state when invoked', () => {
      function CounterView() {
        const count = useCounterSelector((s) => s.count);
        const { increment } = useCounterActions();
        return (
          <div>
            <span data-testid="count">{count}</span>
            <button data-testid="inc-btn" onClick={increment}>
              Increment
            </button>
          </div>
        );
      }

      render(
        <CounterProvider initialCount={0}>
          <CounterView />
        </CounterProvider>
      );

      expect(screen.getByTestId('count').textContent).toBe('0');
      act(() => {
        screen.getByTestId('inc-btn').click();
      });
      expect(screen.getByTestId('count').textContent).toBe('1');
    });

    it('Then callers of useActions do NOT re-render when state changes', () => {
      const actionRenderSpy = vi.fn();
      const stateRenderSpy = vi.fn();

      function ActionButton() {
        actionRenderSpy();
        const { increment } = useCounterActions();
        return (
          <button data-testid="inc-btn" onClick={increment}>
            Increment
          </button>
        );
      }

      function Display() {
        stateRenderSpy();
        const count = useCounterSelector((s) => s.count);
        return <span data-testid="count">{count}</span>;
      }

      render(
        <CounterProvider initialCount={0}>
          <ActionButton />
          <Display />
        </CounterProvider>
      );

      expect(actionRenderSpy).toHaveBeenCalledTimes(1);
      expect(stateRenderSpy).toHaveBeenCalledTimes(1);

      act(() => {
        screen.getByTestId('inc-btn').click();
      });

      // Display re-renders because count changed
      expect(stateRenderSpy).toHaveBeenCalledTimes(2);
      expect(screen.getByTestId('count').textContent).toBe('1');

      // ActionButton must NOT re-render (stable action reference)
      expect(actionRenderSpy).toHaveBeenCalledTimes(1);
    });

    it('Then selectors only re-render when their specific slice changes', () => {
      const textRenderSpy = vi.fn();

      function TextView() {
        textRenderSpy();
        const text = useCounterSelector((s) => s.text);
        return <span data-testid="text">{text}</span>;
      }

      function ActionControls() {
        const { increment, setText } = useCounterActions();
        return (
          <div>
            <button data-testid="inc-btn" onClick={increment}>
              Inc
            </button>
            <button data-testid="set-text-btn" onClick={() => setText('updated')}>
              SetText
            </button>
          </div>
        );
      }

      render(
        <CounterProvider initialCount={0}>
          <TextView />
          <ActionControls />
        </CounterProvider>
      );

      expect(textRenderSpy).toHaveBeenCalledTimes(1);

      // Incrementing count should NOT re-render TextView
      act(() => {
        screen.getByTestId('inc-btn').click();
      });
      expect(textRenderSpy).toHaveBeenCalledTimes(1);

      // Updating text SHOULD re-render TextView
      act(() => {
        screen.getByTestId('set-text-btn').click();
      });
      expect(textRenderSpy).toHaveBeenCalledTimes(2);
      expect(screen.getByTestId('text').textContent).toBe('updated');
    });

    it('Then two sibling providers maintain completely isolated states', () => {
      function InstanceView({ id }: { id: string }) {
        const count = useCounterSelector((s) => s.count);
        const { increment } = useCounterActions();
        return (
          <div>
            <span data-testid={`count-${id}`}>{count}</span>
            <button data-testid={`inc-${id}`} onClick={increment}>
              Inc {id}
            </button>
          </div>
        );
      }

      render(
        <div>
          <CounterProvider initialCount={10}>
            <InstanceView id="first" />
          </CounterProvider>
          <CounterProvider initialCount={50}>
            <InstanceView id="second" />
          </CounterProvider>
        </div>
      );

      expect(screen.getByTestId('count-first').textContent).toBe('10');
      expect(screen.getByTestId('count-second').textContent).toBe('50');

      act(() => {
        screen.getByTestId('inc-first').click();
      });

      expect(screen.getByTestId('count-first').textContent).toBe('11');
      expect(screen.getByTestId('count-second').textContent).toBe('50');
    });

    it('Then changing Provider key creates a fresh isolated store instance', () => {
      function InstanceView() {
        const count = useCounterSelector((s) => s.count);
        const { increment } = useCounterActions();
        return (
          <div>
            <span data-testid="count">{count}</span>
            <button data-testid="inc-btn" onClick={increment}>
              Inc
            </button>
          </div>
        );
      }

      function AppWrapper({ id, initialCount }: { id: string; initialCount: number }) {
        return (
          <CounterProvider key={id} initialCount={initialCount}>
            <InstanceView />
          </CounterProvider>
        );
      }

      const { rerender } = render(<AppWrapper id="app-1" initialCount={10} />);
      expect(screen.getByTestId('count').textContent).toBe('10');

      act(() => {
        screen.getByTestId('inc-btn').click();
      });
      expect(screen.getByTestId('count').textContent).toBe('11');

      // Re-rendering with a new key (e.g. navigating to different app) re-initializes clean store
      rerender(<AppWrapper id="app-2" initialCount={50} />);
      expect(screen.getByTestId('count').textContent).toBe('50');
    });

    it('Then useStore exposes the underlying vanilla store instance', () => {
      function RawInspector() {
        const store = useCounterStore();
        return <div data-testid="raw">{store.getState().count}</div>;
      }

      render(
        <CounterProvider initialCount={42}>
          <RawInspector />
        </CounterProvider>
      );

      expect(screen.getByTestId('raw').textContent).toBe('42');
    });
  });

  describe('When hooks are used outside their Provider', () => {
    it('Then useSelector throws an informative error', () => {
      function OrphanComponent() {
        useCounterSelector((s) => s.count);
        return null;
      }

      expect(() => render(<OrphanComponent />)).toThrow(
        /must be used within its Provider/i
      );
    });

    it('Then useActions throws an informative error', () => {
      function OrphanComponent() {
        useCounterActions();
        return null;
      }

      expect(() => render(<OrphanComponent />)).toThrow(
        /must be used within its Provider/i
      );
    });
  });
});
