import { useState } from 'react';
import {
  X,
  ShieldCheck,
  AlertTriangle,
  AlertCircle,
  CheckCircle2,
  HelpCircle,
  ChevronDown,
  ChevronUp,
  Cpu,
} from 'lucide-react';
import type { CompatibilityReport, FeatureFinding } from '../../utils/compatibility-linter';

export interface CompatibilityReportModalProps {
  isOpen: boolean;
  onClose: () => void;
  report: CompatibilityReport;
  title?: string;
  subtitle?: string;
}

export function CompatibilityReportModal({
  isOpen,
  onClose,
  report,
  title = 'Celld Compatibility Report',
  subtitle = 'Pre-flight evaluation for Cloudflare Workers & Durable Objects migration to Celld',
}: CompatibilityReportModalProps) {
  const [showSupported, setShowSupported] = useState(false);

  if (!isOpen) return null;

  const getStatusBadge = () => {
    switch (report.level) {
      case 'compatible':
        return (
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-emerald-950/60 border border-emerald-800 text-emerald-400 text-xs font-semibold">
            <ShieldCheck className="w-4 h-4 text-emerald-400" />
            <span>Celld Compatible • Safe to Deploy</span>
          </div>
        );
      case 'warning':
        return (
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-amber-950/60 border border-amber-800 text-amber-400 text-xs font-semibold">
            <AlertTriangle className="w-4 h-4 text-amber-400" />
            <span>Deployable with Warnings</span>
          </div>
        );
      case 'incompatible':
        return (
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-rose-950/60 border border-rose-800 text-rose-400 text-xs font-semibold">
            <AlertCircle className="w-4 h-4 text-rose-400" />
            <span>Action Required • Incompatible Features</span>
          </div>
        );
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl w-full max-w-2xl max-h-[90vh] flex flex-col shadow-2xl overflow-hidden">
        {/* Header */}
        <div className="p-4 border-b border-zinc-800 flex items-center justify-between shrink-0">
          <div className="flex items-center gap-2.5">
            <div className="p-2 rounded-lg bg-purple-950/50 border border-purple-800/60 text-purple-400">
              <Cpu className="w-4 h-4" />
            </div>
            <div>
              <h3 className="text-sm font-bold text-zinc-100">{title}</h3>
              <p className="text-xs text-zinc-400">{subtitle}</p>
            </div>
          </div>
          <button
            onClick={onClose}
            aria-label="Close"
            className="p-1.5 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800 transition"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Content Body */}
        <div className="p-4 overflow-y-auto space-y-4 flex-1">
          {/* Status and Summary Card */}
          <div className="p-4 rounded-xl bg-zinc-950/70 border border-zinc-800/80 space-y-3">
            <div className="flex items-center justify-between flex-wrap gap-2">
              {getStatusBadge()}
              <div className="flex items-center gap-3 text-xs">
                <span className="text-emerald-400 font-mono font-semibold">
                  {report.supportedCount} Supported
                </span>
                <span className="text-amber-400 font-mono font-semibold">
                  {report.warningCount} Warnings
                </span>
                <span className="text-rose-400 font-mono font-semibold">
                  {report.unsupportedCount} Incompatible
                </span>
              </div>
            </div>
            <p className="text-xs text-zinc-300 leading-relaxed font-mono">
              {report.summary}
            </p>
          </div>

          {/* Incompatible Features List */}
          {report.unsupportedFeatures.length > 0 && (
            <div className="space-y-2">
              <h4 className="text-xs font-bold text-rose-400 flex items-center gap-1.5">
                <AlertCircle className="w-3.5 h-3.5" />
                Incompatible Features ({report.unsupportedFeatures.length})
              </h4>
              <div className="space-y-2.5">
                {report.unsupportedFeatures.map((item, idx) => (
                  <FindingCard key={idx} finding={item} />
                ))}
              </div>
            </div>
          )}

          {/* Warnings List */}
          {report.warnings.length > 0 && (
            <div className="space-y-2">
              <h4 className="text-xs font-bold text-amber-400 flex items-center gap-1.5">
                <AlertTriangle className="w-3.5 h-3.5" />
                Warnings & Emulations ({report.warnings.length})
              </h4>
              <div className="space-y-2.5">
                {report.warnings.map((item, idx) => (
                  <FindingCard key={idx} finding={item} />
                ))}
              </div>
            </div>
          )}

          {/* Supported Features (Collapsible) */}
          <div className="pt-2 border-t border-zinc-800/60">
            <button
              onClick={() => setShowSupported(!showSupported)}
              className="w-full flex items-center justify-between text-xs font-medium text-zinc-400 hover:text-zinc-200 py-1"
            >
              <span className="flex items-center gap-1.5">
                <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />
                Celld Native Supported Features ({report.supportedCount})
              </span>
              {showSupported ? (
                <ChevronUp className="w-3.5 h-3.5" />
              ) : (
                <ChevronDown className="w-3.5 h-3.5" />
              )}
            </button>

            {showSupported && (
              <div className="mt-2.5 grid grid-cols-1 sm:grid-cols-2 gap-2 animate-in fade-in duration-150">
                {report.supportedFeatures.map((item, idx) => (
                  <div
                    key={idx}
                    className="p-2.5 rounded-lg bg-zinc-950/50 border border-zinc-800/50 flex flex-col justify-between"
                  >
                    <div className="flex items-center gap-2">
                      <CheckCircle2 className="w-3 h-3 text-emerald-400 shrink-0" />
                      <span className="text-xs font-semibold text-zinc-200 truncate">
                        {item.name}
                      </span>
                    </div>
                    <p className="text-[11px] text-zinc-400 mt-1 pl-5">
                      {item.details}
                    </p>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="p-4 border-t border-zinc-800 flex justify-end shrink-0 bg-zinc-900/50">
          <button
            onClick={onClose}
            className="px-4 py-2 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-xs font-medium text-zinc-200 transition"
          >
            Close Report
          </button>
        </div>
      </div>
    </div>
  );
}

function FindingCard({ finding }: { finding: FeatureFinding }) {
  const isUnsupported = finding.status === 'unsupported';

  return (
    <div
      className={`p-3 rounded-lg border ${
        isUnsupported
          ? 'bg-rose-950/20 border-rose-900/50'
          : 'bg-amber-950/20 border-amber-900/50'
      }`}
    >
      <div className="flex items-start justify-between gap-2">
        <div>
          <span
            className={`text-xs font-bold ${
              isUnsupported ? 'text-rose-300' : 'text-amber-300'
            }`}
          >
            {finding.name}
          </span>
          <span className="ml-2 text-[10px] font-mono px-1.5 py-0.5 rounded bg-zinc-800 text-zinc-400">
            {finding.category}
          </span>
        </div>
        {finding.file && (
          <span className="text-[10px] font-mono text-zinc-500">
            {finding.file}
            {finding.line ? `:${finding.line}` : ''}
          </span>
        )}
      </div>

      <p className="text-xs text-zinc-300 mt-1 leading-normal">{finding.details}</p>

      {finding.remediation && (
        <div className="mt-2.5 p-2 rounded bg-zinc-950/80 border border-zinc-800 text-[11px] flex gap-2 items-start">
          <HelpCircle className="w-3.5 h-3.5 text-purple-400 shrink-0 mt-0.5" />
          <div className="text-zinc-300">
            <span className="font-semibold text-purple-300">Remediation: </span>
            {finding.remediation}
          </div>
        </div>
      )}
    </div>
  );
}
