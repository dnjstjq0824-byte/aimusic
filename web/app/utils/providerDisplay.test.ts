import { describe, expect, it } from 'vitest';
import { providerDisplayName } from './providerDisplay';

describe('providerDisplayName (H-04)', () => {
  it('maps known codes to display names', () => {
    expect(providerDisplayName('SUNO')).toBe('Suno');
    expect(providerDisplayName('STABLE_AUDIO')).toBe('Stable Audio');
    expect(providerDisplayName('ELEVENLABS')).toBe('ElevenLabs');
    expect(providerDisplayName('CUSTOM_OPEN_MODEL')).toBe('Custom Open Model');
  });

  it('renders unknown codes as-is, never blank', () => {
    expect(providerDisplayName('FUTURE_AI')).toBe('FUTURE_AI');
    expect(providerDisplayName(null)).toBe('');
    expect(providerDisplayName(undefined)).toBe('');
  });
});
