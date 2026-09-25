import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, fireEvent, cleanup } from '@testing-library/react';
import { CompatibilityReportModal } from './CompatibilityReportModal';
import type { CompatibilityReport } from '../../utils/compatibility-linter';

describe('CompatibilityReportModal', () => {
  afterEach(() => {
    cleanup();
  });
  const mockReport: CompatibilityReport = {
    level: 'incompatible',
    canDeploy: false,
    totalIssues: 2,
    unsupportedCount: 1,
    warningCount: 1,
    supportedCount: 3,
    supportedFeatures: [
      {
        name: 'KV Namespace (MY_KV)',
        category: 'binding',
        status: 'supported',
        details: 'Backed by SQLite key-value store',
      },
    ],
    unsupportedFeatures: [
      {
        name: 'Cloudflare Workers AI (env.AI)',
        category: 'api_usage',
        status: 'unsupported',
        file: 'src/index.ts',
        line: 12,
        details: 'Workers AI model inference (@cf/*) is not natively embedded in celld.',
        remediation:
          'Proxy calls to an external OpenAI/Ollama compatible endpoint or self-hosted LLM server via fetch().',
      },
    ],
    warnings: [
      {
        name: 'Queue Producers / Consumers',
        category: 'config',
        status: 'warning',
        details: 'Queues require external message broker or DO emulation.',
        remediation: 'Queue messages can be dispatched using SQLite Durable Objects.',
      },
    ],
    summary: 'Project has 1 incompatible Cloudflare feature(s) requiring remediation.',
  };

  it('renders modal with incompatible status, warnings, and remediation guidance', () => {
    const handleClose = vi.fn();
    render(
      <CompatibilityReportModal
        isOpen={true}
        onClose={handleClose}
        report={mockReport}
      />
    );

    expect(screen.getByText('Celld Compatibility Report')).toBeDefined();
    expect(screen.getByText('Action Required • Incompatible Features')).toBeDefined();
    expect(screen.getByText('1 Incompatible')).toBeDefined();
    expect(screen.getByText('1 Warnings')).toBeDefined();

    // Incompatible feature details and remediation
    expect(screen.getByText('Cloudflare Workers AI (env.AI)')).toBeDefined();
    expect(
      screen.getByText(/Proxy calls to an external OpenAI\/Ollama/)
    ).toBeDefined();

    // Warning details
    expect(screen.getByText('Queue Producers / Consumers')).toBeDefined();

    // Close button triggers onClose callback
    fireEvent.click(screen.getByText('Close Report'));
    expect(handleClose).toHaveBeenCalledTimes(1);
  });

  it('renders compatible status and expands supported features when clicked', () => {
    const compatibleReport: CompatibilityReport = {
      level: 'compatible',
      canDeploy: true,
      totalIssues: 0,
      unsupportedCount: 0,
      warningCount: 0,
      supportedCount: 2,
      supportedFeatures: [
        {
          name: 'D1 Database (DB)',
          category: 'binding',
          status: 'supported',
          details: 'Backed by private SQLite database',
        },
      ],
      unsupportedFeatures: [],
      warnings: [],
      summary: 'Project is fully compatible with celld.',
    };

    render(
      <CompatibilityReportModal
        isOpen={true}
        onClose={vi.fn()}
        report={compatibleReport}
      />
    );

    expect(screen.getByText('Celld Compatible • Safe to Deploy')).toBeDefined();

    // Click to toggle supported features
    const toggleBtn = screen.getByText(/Celld Native Supported Features/);
    fireEvent.click(toggleBtn);
    expect(screen.getByText('D1 Database (DB)')).toBeDefined();
    expect(screen.getByText('Backed by private SQLite database')).toBeDefined();
  });
});
