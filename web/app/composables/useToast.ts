import { ref } from 'vue';

export interface ToastItem {
  id: string;
  type: 'success' | 'error' | 'warning' | 'info';
  title?: string;
  message: string;
  duration?: number;
}

const toasts = ref<ToastItem[]>([]);

export function useToast() {
  function show(toast: Omit<ToastItem, 'id'>) {
    const id = Math.random().toString(36).substring(2, 9);
    const item: ToastItem = {
      id,
      duration: 4000,
      ...toast,
    };
    toasts.value.push(item);

    if (item.duration && item.duration > 0) {
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

  return {
    toasts,
    show,
    success,
    error,
    warning,
    info,
    dismiss,
  };
}
