// Style-tag option groups for the studio generation panel (WI-G5).
// Display names reuse the tags.* namespace (R5: English keys + 5 locales),
// so no new tag strings are needed here.
export interface StyleTagGroup {
  id: string;
  keys: string[];
}

export const STYLE_TAG_GROUPS: StyleTagGroup[] = [
  { id: 'mood', keys: ['focus', 'chill', 'cafe', 'cinematic', 'ambient'] },
  { id: 'genre', keys: ['electronic', 'pop', 'rock', 'hiphop', 'jazz', 'classical'] },
  { id: 'use', keys: ['work', 'workout', 'drive', 'sleep', 'party', 'youtube'] },
];

export const MAX_STYLE_TAGS = 6;

// toggleTag adds or removes a tag key, capping the selection at max.
// Pure: covered by styleTags.test.ts.
export function toggleTag(selected: string[], key: string, max: number = MAX_STYLE_TAGS): string[] {
  if (selected.includes(key)) {
    return selected.filter((k) => k !== key);
  }
  if (selected.length >= max) {
    return selected;
  }
  return [...selected, key];
}
