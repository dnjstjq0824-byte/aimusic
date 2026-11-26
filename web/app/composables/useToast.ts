import type { Ref } from 'vue';

export interface ToastItem {
  id: string;
  type: 'success' | 'error' | 'warning' | 'info';
  title?: string;
  message: string;
  duration?: number;
}

export interface ToastMethods {
  show(toast: Omit<ToastItem, 'id'>): void;
  success(message: string, title?: string): void;
  error(message: string, title?: string): void;
  warning(message: string, title?: string): void;
  info(message: string, title?: string): void;
  dismiss(id: string): void;
}

// bindToastMethods attaches behavior to any toast list. Purely Vue-reactive
// (no Nuxt APIs), so unit tests drive it with a plain ref.
export function bindToastMethods(toasts: Ref<ToastItem[]>): ToastMethods {
  function show(toast: Omit<ToastItem, 'id'>) {
    const id = Math.random().toString(36).substring(2, 9);
    const item: ToastItem = {
      id,
      duration: 4000,
      ...toast,
    };
    toasts.value.push(item);

    if (typeof window !== 'undefined' && item.duration && item.duration > 0) {
      setTimeout(() => {
        dismiss(id);
      }, item.duration);
    }
  }

  function success(message: string, title?: string) {
    show({ type: 'success', message, title });
  }

  function error(message: string, title?: string) {
    show({ type: 'error', message, title });
  }

  function warning(message: string, title?: string) {
    show({ type: 'warning', message, title });
  }

  function info(message: string, title?: string) {
    show({ type: 'info', message, title });
  }

  function dismiss(id: string) {
    toasts.value = toasts.value.filter((t) => t.id !== id);
  }

  return { show, success, error, warning, info, dismiss };
}

export function useToast(): { toasts: Ref<ToastItem[]> } & ToastMethods {
  // Request-scoped on the server: a module-level ref would leak toasts
  // across SSR requests sharing the process.
  const toasts = useState<ToastItem[]>('app-toasts', () => []);
  return { toasts, ...bindToastMethods(toasts) };
}
