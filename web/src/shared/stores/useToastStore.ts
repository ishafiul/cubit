import { create } from 'zustand';

export type ToastVariant = 'success' | 'error' | 'warning' | 'info';

export interface Toast {
  id: string;
  title: string;
  description?: string;
  variant: ToastVariant;
  duration?: number;
}

export interface ToastState {
  toasts: Toast[];
}

export interface ToastActions {
  addToast: (toast: Omit<Toast, 'id'>) => string;
  dismissToast: (id: string) => void;
  clearToasts: () => void;
}

export type ToastStore = ToastState & { actions: ToastActions };

let toastCounter = 0;

export const useToastStore = create<ToastStore>((set, get) => {
  const actions: ToastActions = {
    addToast: (toastInput) => {
      const id = `toast-${Date.now()}-${++toastCounter}`;
      const toast: Toast = {
        ...toastInput,
        id,
        duration: toastInput.duration ?? 4000,
      };

      set((state) => ({
        toasts: [...state.toasts, toast],
      }));

      if (toast.duration && toast.duration > 0) {
        setTimeout(() => {
          get().actions.dismissToast(id);
        }, toast.duration);
      }

      return id;
    },

    dismissToast: (id) => {
      set((state) => ({
        toasts: state.toasts.filter((t) => t.id !== id),
      }));
    },

    clearToasts: () => {
      set({ toasts: [] });
    },
  };

  return {
    toasts: [],
    actions,
  };
});

export function useToastActions(): ToastActions {
  return useToastStore((state) => state.actions);
}

export function useToasts(): Toast[] {
  return useToastStore((state) => state.toasts);
}
