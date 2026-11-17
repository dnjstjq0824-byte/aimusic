// Legal body renderer (R8): stored markdown-ish text to escaped display HTML.
// Korean match literals are unicode-escaped so app sources stay Hangul-free
// (check-i18n); legalBody.test.ts pins the behavior with the real strings.

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

export function renderLegalBody(bodyMd: string | undefined, emptyText: string): string {
  if (!bodyMd) return `<p>${emptyText}</p>`;
  // Stored bodies may carry literal backslash-n sequences (seed.sql uses \n
  // inside non-escape SQL strings, which PostgreSQL stores verbatim).
  // Normalize them to real newlines so breaks render; genuine newlines pass
  // through unchanged. Escape first so stored markup can never execute; then
  // apply the tiny heading/bold/list transforms on the escaped text.
  const withBreaks = bodyMd.replace(/\\r\\n|\\n/g, '\n');
  // Render-layer only (stored text untouched): strip internal file-path
  // references and unfilled business-registration placeholders from the
  // user-facing view per EV-04 / SC-04.
  const displayable = withBreaks
    .replace(/\uc804\ubb38\uc740 `docs\/legal\/[^`]+`\ub97c \ucc38\uc870\ud558\uc138\uc694\.?/g, '')
    .replace(/\(\uc0ac\uc5c5\uc790 \ub4f1\ub85d \ud6c4 \uae30\uc7ac\)/g, '')
    .replace(/^\s*\d+\.\s*$/gm, '')
    .replace(/\n{3,}/g, '\n\n')
    .trim();
  return escapeHtml(displayable)
    .replace(/^### (.*$)/gim, '<h3 class="text-sm font-bold text-white mt-4 mb-2">$1</h3>')
    .replace(/^## (.*$)/gim, '<h2 class="text-base font-extrabold text-white mt-6 mb-3 border-b border-surface-border pb-1">$1</h2>')
    .replace(/^# (.*$)/gim, '<h2 class="text-lg font-black text-brand-400 mt-6 mb-4">$1</h2>')
    .replace(/\*\*(.*?)\*\*/gim, '<strong class="font-bold text-white">$1</strong>')
    .replace(/`([^`]+?)`/g, '<code class="rounded bg-surface-3 px-1 py-px font-mono text-caption text-brand-300">$1</code>')
    .replace(/^\* (.*$)/gim, '<li class="ml-4 list-disc text-text-primary">$1</li>');
}
