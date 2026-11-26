// AI model directory metadata (no content import: safe to use from any page).
export type ModelStatus = 'own' | 'accepted' | 'blocked';

export interface ModelMeta {
  slug: string;
  /** Engine provider code (tracks.ai_provider). */
  code: string;
  name: string;
  status: ModelStatus;
}

// Mirrors the engine license gate (ai_provider_policies, policy 2026-11-11).
export const MODEL_METAS: ModelMeta[] = [
  { slug: 'avibe', code: 'AVIBE', name: 'AVIBE Model', status: 'own' },
  { slug: 'suno', code: 'SUNO', name: 'Suno', status: 'accepted' },
  { slug: 'udio', code: 'UDIO', name: 'Udio', status: 'accepted' },
  { slug: 'aiva', code: 'AIVA', name: 'AIVA', status: 'accepted' },
  { slug: 'stable-audio', code: 'STABLE_AUDIO', name: 'Stable Audio', status: 'accepted' },
  { slug: 'elevenlabs', code: 'ELEVENLABS', name: 'ElevenLabs Music', status: 'accepted' },
  { slug: 'open-model', code: 'CUSTOM_OPEN_MODEL', name: 'Open models', status: 'accepted' },
  { slug: 'soundraw', code: 'SOUNDRAW', name: 'Soundraw', status: 'blocked' },
  { slug: 'mubert', code: 'MUBERT', name: 'Mubert', status: 'blocked' },
];

export function modelBySlug(slug: string): ModelMeta | undefined {
  return MODEL_METAS.find((m) => m.slug === slug);
}

export function modelSlugForProvider(code: string | null | undefined): string | null {
  if (!code) return null;
  return MODEL_METAS.find((m) => m.code === code)?.slug ?? null;
}
