import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, act, cleanup } from '@testing-library/react';
import {
  ApplicationStoreProvider,
  useApplicationSelector,
  useApplicationActions,
  useActiveTab,
  useIsDirty,
  useDraftCode,
  useSecretVisibility,
} from './applicationStore';

afterEach(() => {
  cleanup();
});

describe('Given the scoped applicationStore', () => {
  describe('When rendered inside ApplicationStoreProvider', () => {
    it('Then initializes with props provided to provider', () => {
      function TestComponent() {
        const appId = useApplicationSelector((s) => s.appId);
        const activeTab = useActiveTab();
        const draftCode = useDraftCode();
        const isDirty = useIsDirty();

        return (
          <div>
            <span data-testid="app-id">{appId}</span>
            <span data-testid="tab">{activeTab}</span>
            <span data-testid="code">{draftCode}</span>
            <span data-testid="dirty">{isDirty ? 'dirty' : 'clean'}</span>
          </div>
        );
      }

      render(
        <ApplicationStoreProvider
          appId="app-1"
          initialTab="code"
          initialCode="console.log('hello');"
        >
          <TestComponent />
        </ApplicationStoreProvider>
      );

      expect(screen.getByTestId('app-id').textContent).toBe('app-1');
      expect(screen.getByTestId('tab').textContent).toBe('code');
      expect(screen.getByTestId('code').textContent).toBe("console.log('hello');");
      expect(screen.getByTestId('dirty').textContent).toBe('clean');
    });

    it('Then updating draft code marks isDirty true', () => {
      function Editor() {
        const isDirty = useIsDirty();
        const { setDraftCode } = useApplicationActions();

        return (
          <div>
            <span data-testid="dirty">{isDirty ? 'dirty' : 'clean'}</span>
            <button
              data-testid="edit-btn"
              onClick={() => setDraftCode("console.log('modified');")}
            >
              Edit
            </button>
          </div>
        );
      }

      render(
        <ApplicationStoreProvider appId="app-1" initialCode="original">
          <Editor />
        </ApplicationStoreProvider>
      );

      expect(screen.getByTestId('dirty').textContent).toBe('clean');

      act(() => {
        screen.getByTestId('edit-btn').click();
      });

      expect(screen.getByTestId('dirty').textContent).toBe('dirty');
    });

    it('Then markCodeSaved resets isDirty to false with new saved code', () => {
      function Editor() {
        const isDirty = useIsDirty();
        const { setDraftCode, markCodeSaved } = useApplicationActions();

        return (
          <div>
            <span data-testid="dirty">{isDirty ? 'dirty' : 'clean'}</span>
            <button
              data-testid="edit-btn"
              onClick={() => setDraftCode("console.log('modified');")}
            >
              Edit
            </button>
            <button
              data-testid="save-btn"
              onClick={() => markCodeSaved("console.log('modified');")}
            >
              Save
            </button>
          </div>
        );
      }

      render(
        <ApplicationStoreProvider appId="app-1" initialCode="original">
          <Editor />
        </ApplicationStoreProvider>
      );

      act(() => {
        screen.getByTestId('edit-btn').click();
      });
      expect(screen.getByTestId('dirty').textContent).toBe('dirty');

      act(() => {
        screen.getByTestId('save-btn').click();
      });
      expect(screen.getByTestId('dirty').textContent).toBe('clean');
    });

    it('Then resetDraftCode restores the original saved code and clears isDirty', () => {
      function Editor() {
        const draftCode = useDraftCode();
        const isDirty = useIsDirty();
        const { setDraftCode, resetDraftCode } = useApplicationActions();

        return (
          <div>
            <span data-testid="code">{draftCode}</span>
            <span data-testid="dirty">{isDirty ? 'dirty' : 'clean'}</span>
            <button data-testid="edit-btn" onClick={() => setDraftCode('changed')}>
              Edit
            </button>
            <button data-testid="reset-btn" onClick={resetDraftCode}>
              Reset
            </button>
          </div>
        );
      }

      render(
        <ApplicationStoreProvider appId="app-1" initialCode="original">
          <Editor />
        </ApplicationStoreProvider>
      );

      act(() => {
        screen.getByTestId('edit-btn').click();
      });
      expect(screen.getByTestId('code').textContent).toBe('changed');
      expect(screen.getByTestId('dirty').textContent).toBe('dirty');

      act(() => {
        screen.getByTestId('reset-btn').click();
      });
      expect(screen.getByTestId('code').textContent).toBe('original');
      expect(screen.getByTestId('dirty').textContent).toBe('clean');
    });

    it('Then consumers of useApplicationActions do not re-render on state changes', () => {
      const actionRenderSpy = vi.fn();

      function ActionButton() {
        actionRenderSpy();
        const { setActiveTab } = useApplicationActions();
        return (
          <button data-testid="tab-btn" onClick={() => setActiveTab('settings')}>
            Settings
          </button>
        );
      }

      function TabDisplay() {
        const activeTab = useActiveTab();
        return <span data-testid="active-tab">{activeTab}</span>;
      }

      render(
        <ApplicationStoreProvider appId="app-1" initialTab="overview">
          <ActionButton />
          <TabDisplay />
        </ApplicationStoreProvider>
      );

      expect(actionRenderSpy).toHaveBeenCalledTimes(1);

      act(() => {
        screen.getByTestId('tab-btn').click();
      });

      expect(screen.getByTestId('active-tab').textContent).toBe('settings');
      expect(actionRenderSpy).toHaveBeenCalledTimes(1);
    });

    it('Then toggling secret visibility updates only the targeted key', () => {
      function SecretViewer({ secretKey }: { secretKey: string }) {
        const isVisible = useSecretVisibility(secretKey);
        const { toggleSecretVisibility } = useApplicationActions();

        return (
          <div>
            <span data-testid={`vis-${secretKey}`}>{isVisible ? 'visible' : 'hidden'}</span>
            <button
              data-testid={`toggle-${secretKey}`}
              onClick={() => toggleSecretVisibility(secretKey)}
            >
              Toggle
            </button>
          </div>
        );
      }

      render(
        <ApplicationStoreProvider appId="app-1">
          <SecretViewer secretKey="API_KEY" />
          <SecretViewer secretKey="DB_PASS" />
        </ApplicationStoreProvider>
      );

      expect(screen.getByTestId('vis-API_KEY').textContent).toBe('hidden');
      expect(screen.getByTestId('vis-DB_PASS').textContent).toBe('hidden');

      act(() => {
        screen.getByTestId('toggle-API_KEY').click();
      });

      expect(screen.getByTestId('vis-API_KEY').textContent).toBe('visible');
      expect(screen.getByTestId('vis-DB_PASS').textContent).toBe('hidden');
    });
  });
});
