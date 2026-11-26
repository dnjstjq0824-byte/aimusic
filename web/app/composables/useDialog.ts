// Shared dialog behaviors (P1): nested-safe body scroll lock, focus trap,
// and focusable collection. Used by UiModal, BottomSheet, QueueDrawer, and
// FullscreenPlayer so nested overlays never fight over scroll or focus.

export function scrollLocks(): number {
  return Number(document.body.dataset.modalLocks ?? '0');
}

export function lockBodyScroll(): void {
  const next = scrollLocks() + 1;
  document.body.dataset.modalLocks = String(next);
  if (next === 1) {
    document.body.style.overflow = 'hidden';
  }
}

export function unlockBodyScroll(): void {
  const next = Math.max(0, scrollLocks() - 1);
  document.body.dataset.modalLocks = String(next);
  if (next === 0) {
    document.body.style.overflow = '';
  }
}

export function focusablesIn(panel: HTMLElement | null): HTMLElement[] {
  if (!panel) return [];
  const nodes = panel.querySelectorAll<HTMLElement>(
    'a[href], button:not([disabled]), textarea, input, select, [tabindex]:not([tabindex="-1"])',
  );
  return Array.from(nodes).filter((el) => !el.hasAttribute('disabled') && el.offsetParent !== null);
}

export function trapTabIn(panel: HTMLElement | null, e: KeyboardEvent): void {
  const items = focusablesIn(panel);
  if (items.length === 0) {
    e.preventDefault();
    panel?.focus();
    return;
  }
  const first = items[0] as HTMLElement;
  const last = items[items.length - 1] as HTMLElement;
  if (e.shiftKey && document.activeElement === first) {
    e.preventDefault();
    last.focus();
  } else if (!e.shiftKey && document.activeElement === last) {
    e.preventDefault();
    first.focus();
  }
}
