// AI provider code -> display name (R5 H-04). Codes come from the engine
// (license policies); unknown codes render as-is, never blank.
const DISPLAY: Record<string, string> = {
  SUNO: 'Suno',
  UDIO: 'Udio',
  AIVA: 'AIVA',
  STABLE_AUDIO: 'Stable Audio',
  SOUNDRAW: 'Soundraw',
  MUBERT: 'Mubert',
  ELEVENLABS: 'ElevenLabs',
  CUSTOM_OPEN_MODEL: 'Custom Open Model',
};

export function providerDisplayName(code: string | null | undefined): string {
  if (!code) return '';
  return DISPLAY[code] ?? code;
}
