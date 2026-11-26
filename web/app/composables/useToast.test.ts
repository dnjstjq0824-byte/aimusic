import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ref } from 'vue';
import { bindToastMethods, type ToastItem } from './useToast';

// NOTE: useToast() itself needs Nuxt (useState) and is covered by typecheck,
// build, and the SSR smoke run. These tests drive the pure core with a local
// ref; separate refs never share state (the old module-level leak).

describe('bindToastMethods', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  function setup() {
    const toasts = ref<ToastItem[]>([]);
    return { toasts, ...bindToastMethods(toasts) };
  }

  it('pushes typed toasts with unique ids', () => {
    const { toasts, success, error } = setup();
    success('saved', 'Done');
    error('failed');
    expect(toasts.value).toHaveLength(2);
    expect(toasts.value[0]).toMatchObject({ type: 'success', message: 'saved', title: 'Done' });
    expect(toasts.value[1]).toMatchObject({ type: 'error', message: 'failed' });
    expect(toasts.value[0]?.id).not.toBe(toasts.value[1]?.id);
  });

  it('dismisses by id', () => {
    const { toasts, info, dismiss } = setup();
    info('hello');
    const id = toasts.value[0]?.id ?? '';
    dismiss(id);
    expect(toasts.value).toHaveLength(0);
  });

  it('auto-dismisses after the duration', () => {
    const { toasts, warning } = setup();
    warning('slow down');
    expect(toasts.value).toHaveLength(1);
    vi.advanceTimersByTime(4000);
    expect(toasts.value).toHaveLength(0);
  });

  it('keeps persistent toasts until dismissed', () => {
    const { toasts, show, dismiss } = setup();
    show({ type: 'info', message: 'sticky', duration: 0 });
    vi.advanceTimersByTime(60000);
    expect(toasts.value).toHaveLength(1);
    dismiss(toasts.value[0]?.id ?? '');
    expect(toasts.value).toHaveLength(0);
  });

  it('isolates separate lists (no cross-request leak)', () => {
    const a = setup();
    const b = setup();
    a.success('only in a');
    expect(a.toasts.value).toHaveLength(1);
    expect(b.toasts.value).toHaveLength(0);
  });
});
