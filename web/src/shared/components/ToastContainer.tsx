import { useToasts, useToastActions } from '../stores/useToastStore';
import { CheckCircle2, AlertCircle, AlertTriangle, Info, X } from 'lucide-react';

export function ToastContainer() {
  const toasts = useToasts();
  const { dismissToast } = useToastActions();

  if (toasts.length === 0) return null;

  return (
    <div
      data-testid="toast-container"
      className="fixed bottom-4 right-4 z-50 flex flex-col gap-2 max-w-sm w-full pointer-events-none"
    >
      {toasts.map((toast) => {
        let borderClass = 'border-zinc-800 bg-zinc-900/95 text-zinc-100';
        let Icon = Info;
        let iconClass = 'text-blue-400';

        if (toast.variant === 'success') {
          borderClass = 'border-emerald-800/80 bg-zinc-900/95 text-zinc-100';
          Icon = CheckCircle2;
          iconClass = 'text-emerald-400';
        } else if (toast.variant === 'error') {
          borderClass = 'border-rose-800/80 bg-zinc-900/95 text-zinc-100';
          Icon = AlertCircle;
          iconClass = 'text-rose-400';
        } else if (toast.variant === 'warning') {
          borderClass = 'border-amber-800/80 bg-zinc-900/95 text-zinc-100';
          Icon = AlertTriangle;
          iconClass = 'text-amber-400';
        }

        return (
          <div
            key={toast.id}
            data-testid={`toast-${toast.id}`}
            className={`pointer-events-auto flex items-start gap-3 p-3.5 rounded-lg border shadow-xl backdrop-blur transition-all duration-200 ${borderClass}`}
          >
            <Icon className={`w-5 h-5 shrink-0 mt-0.5 ${iconClass}`} />
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium leading-tight">{toast.title}</p>
              {toast.description && (
                <p className="text-xs text-zinc-400 mt-1 leading-normal">{toast.description}</p>
              )}
            </div>
            <button
              data-testid={`dismiss-${toast.id}`}
              onClick={() => dismissToast(toast.id)}
              className="text-zinc-500 hover:text-zinc-300 p-0.5 rounded transition-colors"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        );
      })}
    </div>
  );
}
