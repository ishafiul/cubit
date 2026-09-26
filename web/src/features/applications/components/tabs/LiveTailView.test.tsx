import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, act, cleanup } from '@testing-library/react';
import { ApplicationStoreProvider } from '../../stores/applicationStore';
import { LiveTailView } from './LiveTailView';
import type { Application, RequestLogEvent } from '../../../../api/model';

afterEach(() => {
  cleanup();
});

const mockApp: Application = {
  id: 'app-test',
  name: 'test-worker',
  sourceType: 'inline',
  inlineCode: 'export default { fetch() {} }',
  status: 'running',
  createdAt: '2026-09-25T10:00:00Z',
  updatedAt: '2026-09-25T10:00:00Z',
};

const mockEvent: RequestLogEvent = {
  id: 'ray_1234567890',
  timestamp: '2026-09-25T12:00:00Z',
  method: 'GET',
  path: '/api/v1/hello',
  url: 'http://test-worker.localhost:8000/api/v1/hello',
  statusCode: 200,
  durationMs: 14.5,
  clientIp: '192.168.1.50',
  outcome: 'Success',
  requestHeaders: {
    'user-agent': 'Mozilla/5.0',
    accept: 'application/json',
  },
  responseHeaders: {
    'content-type': 'application/json',
  },
  requestBody: '{"query": "ping"}',
  responseBody: '{"reply": "pong"}',
  logs: [
    {
      timestamp: Date.now(),
      level: 'info',
      message: 'Processing ping request',
    },
  ],
  cf: {
    country: 'US',
    city: 'San Francisco',
    colo: 'SFO',
    asn: 13335,
    httpProtocol: 'HTTP/2',
    tlsVersion: 'TLSv1.3',
  },
};

describe('Given LiveTailView component', () => {
  describe('When rendered without logs', () => {
    it('Then renders the toolbar and active waiting state', () => {
      render(
        <ApplicationStoreProvider appId="app-test">
          <LiveTailView app={mockApp} />
        </ApplicationStoreProvider>
      );

      expect(screen.getByTestId('live-tail-view')).toBeDefined();
      expect(screen.getByTestId('tail-status-indicator')).toBeDefined();
      expect(screen.getByText('Live Request Tail Active')).toBeDefined();
    });

    it('Then clicking send test request invokes callback', () => {
      const onTestSpy = vi.fn();
      render(
        <ApplicationStoreProvider appId="app-test">
          <LiveTailView app={mockApp} onTestApp={onTestSpy} />
        </ApplicationStoreProvider>
      );

      const testBtn = screen.getByTestId('tail-send-test-btn');
      expect(testBtn).toBeDefined();

      act(() => {
        testBtn.click();
      });

      expect(onTestSpy).toHaveBeenCalledWith(mockApp);
    });
  });

  describe('When simulated with event stream', () => {
    it('Then handles pausing and resuming stream', () => {
      render(
        <ApplicationStoreProvider appId="app-test">
          <LiveTailView app={mockApp} />
        </ApplicationStoreProvider>
      );

      const pauseBtn = screen.getByTestId('tail-pause-resume-btn');
      expect(pauseBtn.textContent).toContain('Pause Stream');

      act(() => {
        pauseBtn.click();
      });

      expect(pauseBtn.textContent).toContain('Resume Stream');
    });

    it('Then streams live event and renders inspector when row is clicked', () => {
      let instance: any = null;
      class MockEventSource {
        onopen: (() => void) | null = null;
        onmessage: ((e: MessageEvent) => void) | null = null;
        onerror: (() => void) | null = null;
        close = vi.fn();
        constructor() {
          instance = this;
        }
      }
      (globalThis as any).EventSource = MockEventSource;

      render(
        <ApplicationStoreProvider appId="app-test">
          <LiveTailView app={mockApp} />
        </ApplicationStoreProvider>
      );

      act(() => {
        instance.onopen?.();
        instance.onmessage?.({ data: JSON.stringify(mockEvent) });
      });

      expect(screen.getByTestId('log-row-ray_1234567890')).toBeDefined();
      expect(screen.getByText('/api/v1/hello')).toBeDefined();

      act(() => {
        screen.getByTestId('log-row-ray_1234567890').firstElementChild?.dispatchEvent(
          new MouseEvent('click', { bubbles: true })
        );
      });

      expect(screen.getByTestId('log-inspector-drawer')).toBeDefined();
      delete (globalThis as any).EventSource;
    });
  });
});
