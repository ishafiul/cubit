import React, { useRef, useState } from 'react';
import { Copy, Check } from 'lucide-react';
import { computeLineCount, indentCode, outdentCode } from './code-utils';

export interface CodeEditorProps {
  value: string;
  onChange?: (value: string) => void;
  readOnly?: boolean;
  minHeight?: string;
  placeholder?: string;
}

export function CodeEditor({
  value,
  onChange,
  readOnly = false,
  minHeight = '360px',
  placeholder = '// Write your Cloudflare Worker script here...',
}: CodeEditorProps) {
  const [copied, setCopied] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const lineNumbersRef = useRef<HTMLDivElement>(null);

  const lineCount = computeLineCount(value);

  // Sync line number scroll with textarea scroll
  const handleScroll = (e: React.UIEvent<HTMLTextAreaElement>) => {
    if (lineNumbersRef.current) {
      lineNumbersRef.current.scrollTop = e.currentTarget.scrollTop;
    }
  };

  // Intercept Tab key to indent/outdent with 2 spaces
  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (readOnly || !onChange) return;

    if (e.key === 'Tab') {
      e.preventDefault();
      const textarea = e.currentTarget;
      const start = textarea.selectionStart;
      const end = textarea.selectionEnd;

      if (e.shiftKey) {
        const { updatedCode, newStart, newEnd } = outdentCode(value, start, end, 2);
        if (updatedCode !== value) {
          onChange(updatedCode);
          requestAnimationFrame(() => {
            textarea.selectionStart = newStart;
            textarea.selectionEnd = newEnd;
          });
        }
      } else {
        const { updatedCode, newStart, newEnd } = indentCode(value, start, end, 2);
        onChange(updatedCode);
        requestAnimationFrame(() => {
          textarea.selectionStart = newStart;
          textarea.selectionEnd = newEnd;
        });
      }
    }
  };

  const handleCopy = () => {
    navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="flex flex-col rounded-xl border border-zinc-800 bg-zinc-950 overflow-hidden shadow-lg">
      {/* Editor Top Bar */}
      <div className="flex items-center justify-between px-4 py-2 bg-zinc-900/80 border-b border-zinc-800/80 text-xs text-zinc-400">
        <div className="flex items-center gap-2">
          <span className="w-2.5 h-2.5 rounded-full bg-emerald-500/80 inline-block" />
          <span className="font-mono font-medium text-zinc-300">index.js</span>
          <span className="text-[10px] text-zinc-500 font-mono">ES Module (fetch)</span>
        </div>
        <div className="flex items-center gap-3">
          <span className="text-[11px] font-mono text-zinc-500">
            {lineCount} {lineCount === 1 ? 'line' : 'lines'} • {value.length} chars
          </span>
          <button
            type="button"
            onClick={handleCopy}
            title="Copy code"
            className="flex items-center gap-1 text-[11px] text-zinc-400 hover:text-zinc-200 transition p-1 rounded hover:bg-zinc-800"
          >
            {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
            <span>{copied ? 'Copied' : 'Copy'}</span>
          </button>
        </div>
      </div>

      {/* Editor Body: Line numbers + Textarea */}
      <div className="flex relative font-mono text-xs overflow-hidden" style={{ minHeight }}>
        {/* Line Numbers Gutter */}
        <div
          ref={lineNumbersRef}
          aria-hidden="true"
          className="w-12 py-3 pr-3 text-right text-zinc-600 bg-zinc-950 select-none overflow-hidden border-r border-zinc-900 leading-6 shrink-0"
        >
          {Array.from({ length: lineCount }, (_, i) => (
            <div key={i + 1}>{i + 1}</div>
          ))}
        </div>

        {/* Code Input */}
        <textarea
          ref={textareaRef}
          value={value}
          onChange={e => onChange && onChange(e.target.value)}
          onScroll={handleScroll}
          onKeyDown={handleKeyDown}
          readOnly={readOnly}
          spellCheck={false}
          placeholder={placeholder}
          className="flex-1 p-3 bg-transparent text-zinc-100 placeholder-zinc-700 outline-none resize-none leading-6 overflow-y-auto whitespace-pre font-mono selection:bg-emerald-950 selection:text-emerald-200"
        />
      </div>
    </div>
  );
}
