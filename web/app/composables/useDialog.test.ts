import { afterEach, describe, expect, it } from 'vitest';
import {
  focusablesIn,
  lockBodyScroll,
  trapTabIn,
  unlockBodyScroll,
} from './useDialog';

describe('body scroll lock counter', () => {
  afterEach(() => {
    delete document.body.dataset.modalLocks;
    document.body.style.overflow = '';
  });

  it('locks on first lock and releases on last unlock', () => {
    lockBodyScroll();
    expect(document.body.style.overflow).toBe('hidden');
    lockBodyScroll();
    unlockBodyScroll();
    expect(document.body.style.overflow).toBe('hidden');
    unlockBodyScroll();
    expect(document.body.style.overflow).toBe('');
  });

  it('tolerates unbalanced unlocks', () => {
    unlockBodyScroll();
    expect(document.body.style.overflow).toBe('');
    expect(document.body.dataset.modalLocks).toBe('0');
  });
});

describe('focusablesIn', () => {
  it('collects interactive descendants, skipping disabled ones', () => {
    document.body.innerHTML = `<div id="p">
      <button id="a">a</button>
      <button id="b" disabled>b</button>
      <a id="c" href="/x">c</a>
      <span id="d" tabindex="0">d</span>
    </div>`;
    const ids = focusablesIn(document.getElementById('p') as HTMLElement).map((el) => el.id);
    expect(ids).toEqual(['a', 'c', 'd']);
    document.body.innerHTML = '';
  });

  it('returns empty for a null panel', () => {
    expect(focusablesIn(null)).toEqual([]);
  });
});

describe('trapTabIn', () => {
  it('wraps focus from last to first and back', () => {
    document.body.innerHTML = `<div id="p">
      <button id="a">a</button>
      <button id="b">b</button>
    </div>`;
    const panel = document.getElementById('p') as HTMLElement;
    const a = document.getElementById('a') as HTMLElement;
    const b = document.getElementById('b') as HTMLElement;
    const tab = (shift: boolean): KeyboardEvent => {
      const e = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true });
      // jsdom/happy-dom KeyboardEvent init does not take shiftKey reliably; define it.
      Object.defineProperty(e, 'shiftKey', { value: shift });
      return e;
    };
    b.focus();
    const fwd = tab(false);
    trapTabIn(panel, fwd);
    expect(fwd.defaultPrevented).toBe(true);
    expect(document.activeElement).toBe(a);
    const back = tab(true);
    trapTabIn(panel, back);
    expect(back.defaultPrevented).toBe(true);
    expect(document.activeElement).toBe(b);
    document.body.innerHTML = '';
  });

  it('focuses the panel when nothing is focusable', () => {
    document.body.innerHTML = '<div id="p" tabindex="-1"></div>';
    const panel = document.getElementById('p') as HTMLElement;
    panel.focus = (() => {
      (panel as unknown as { focused: boolean }).focused = true;
    }) as unknown as typeof panel.focus;
    const e = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true });
    trapTabIn(panel, e);
    expect(e.defaultPrevented).toBe(true);
    expect((panel as unknown as { focused: boolean }).focused).toBe(true);
    document.body.innerHTML = '';
  });
});
