// Legal term titles per locale (P1): the server stores Korean-only titles
// (titleKo), but UI chrome (consent gate, doc header, privacy list) shows a
// translated title when the locale ships one, falling back to titleKo.

const TERM_TITLE_KEYS: Record<string, string> = {
  TOS: 'legal.docTOS',
  PRIVACY: 'legal.docPRIVACY',
  COPYRIGHT_POLICY: 'legal.docCOPYRIGHT',
  PARTNER_CONTRACT: 'legal.docPARTNER',
};

export function legalTermTitleKey(code: string): string | null {
  return TERM_TITLE_KEYS[code] ?? null;
}

export function resolveTermTitle(
  code: string,
  titleKo: string,
  te: (key: string) => boolean,
  t: (key: string) => string,
): string {
  const key = legalTermTitleKey(code);
  return key !== null && te(key) ? t(key) : titleKo;
}
