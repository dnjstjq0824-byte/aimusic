import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useToast } from './useToast';

describe('useToast', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    const { toasts } = useToast();
    toasts.value = [];
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('pushes typed toasts with unique ids', () => {
    const { toasts, success, error } = useToast();
    success('saved', 'Done');
    error('failed');
    expect(toasts.value).toHaveLength(2);
    expect(toasts.value[0]).toMatchObject({ type: 'success', message: 'saved', title: 'Done' });
    expect(toasts.value[1]).toMatchObject({ type: 'error', message: 'failed' });
    expect(toasts.value[0]?.id).not.toBe(toasts.value[1]?.id);
  });

  it('dismisses by id', () => {
    const { toasts, info, dismiss } = useToast();
    info('hello');
    const id = toasts.value[0]?.id ?? '';
    dismiss(id);
    expect(toasts.value).toHaveLength(0);
  });

  it('auto-dismisses after the duration', () => {
    const { toasts, warning } = useToast();
    warning('slow down');
    expect(toasts.value).toHaveLength(1);
    vi.advanceTimersByTime(4000);
    expect(toasts.value).toHaveLength(0);
  });

  it('keeps persistent toasts until dismissed', () => {
    const { toasts, show, dismiss } = useToast();
    show({ type: 'info', message: 'sticky', duration: 0 });
    vi.advanceTimersByTime(60000);
    expect(toasts.value).toHaveLength(1);
    dismiss(toasts.value[0]?.id ?? '');
    expect(toasts.value).toHaveLength(0);
  });
});
