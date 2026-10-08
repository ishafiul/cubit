import { render, screen, fireEvent, cleanup } from '@testing-library/react';
import { describe, it, expect, vi, afterEach } from 'vitest';
import React from 'react';
import { DynamicWorkersView } from './DynamicWorkersView';

// Mock CodeEditor so we don't need real Monaco/DOM measurements
vi.mock('../CodeEditor', () => ({
  CodeEditor: ({ value, onChange }: { value: string; onChange: (v: string) => void }) => (
    <textarea data-testid="code-editor" value={value} onChange={(e) => onChange(e.target.value)} />
  ),
}));

describe('DynamicWorkersView', () => {
  afterEach(() => {
    cleanup();
  });
  it('Given DynamicWorkersView When rendered Then displays celld 0.6.2 Runtime badge and templates', () => {
    render(<DynamicWorkersView />);

    expect(screen.getByRole('heading', { name: /Dynamic Workers/i })).toBeDefined();
    expect(screen.getByText('celld 0.6.2 Runtime')).toBeDefined();

    // Check default Hello World template contains 0.6.2
    const editor = screen.getByTestId('code-editor') as HTMLTextAreaElement;
    expect(editor.value).toContain('Dynamic Celld 0.6.2 Worker');
  });

  it('Given DynamicWorkersView When selecting timeout/self template Then populates AbortSignal timeout and self globals', () => {
    render(<DynamicWorkersView />);

    const timeoutBtn = screen.getByRole('button', { name: /AbortSignal & self/i });
    fireEvent.click(timeoutBtn);

    const editor = screen.getByTestId('code-editor') as HTMLTextAreaElement;
    expect(editor.value).toContain('AbortSignal.timeout(5000)');
    expect(editor.value).toContain('typeof self !== "undefined"');
    expect(editor.value).toContain('celld_runtime: "v0.6.2"');
  });
});
