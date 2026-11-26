import { describe, expect, it } from 'vitest';
import { nextTabIndex } from './tabs';

describe('nextTabIndex (APG tabs keyboard)', () => {
  it('moves right/down with wraparound', () => {
    expect(nextTabIndex(3, 0, 'ArrowRight')).toBe(1);
    expect(nextTabIndex(3, 2, 'ArrowRight')).toBe(0);
    expect(nextTabIndex(3, 1, 'ArrowDown')).toBe(2);
  });
  it('moves left/up with wraparound', () => {
    expect(nextTabIndex(3, 1, 'ArrowLeft')).toBe(0);
    expect(nextTabIndex(3, 0, 'ArrowLeft')).toBe(2);
    expect(nextTabIndex(3, 2, 'ArrowUp')).toBe(1);
  });
  it('jumps Home/End', () => {
    expect(nextTabIndex(3, 2, 'Home')).toBe(0);
    expect(nextTabIndex(3, 0, 'End')).toBe(2);
  });
  it('ignores other keys and empty lists', () => {
    expect(nextTabIndex(3, 1, 'Enter')).toBeNull();
    expect(nextTabIndex(3, 1, 'Tab')).toBeNull();
    expect(nextTabIndex(0, 0, 'ArrowRight')).toBeNull();
  });
});
