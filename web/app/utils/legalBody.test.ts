import { describe, expect, it } from 'vitest';
import { renderLegalBody } from '@/utils/legalBody';

describe('renderLegalBody', () => {
  it('returns the empty text when the body is missing', () => {
    expect(renderLegalBody(undefined, 'EMPTY')).toBe('<p>EMPTY</p>');
    expect(renderLegalBody('', 'EMPTY')).toBe('<p>EMPTY</p>');
  });

  it('strips internal doc-path references (EV-04)', () => {
    const out = renderLegalBody('제1조 목적\n전문은 `docs/legal/terms.md`를 참조하세요.\n제2조 정의', 'EMPTY');
    expect(out).not.toContain('docs/legal');
    expect(out).toContain('제1조 목적');
    expect(out).toContain('제2조 정의');
  });

  it('strips unfilled business-registration placeholders (SC-04)', () => {
    const out = renderLegalBody('상호: 에이바이브 (사업자 등록 후 기재)', 'EMPTY');
    expect(out).not.toContain('사업자 등록 후 기재');
    expect(out).toContain('상호');
  });

  it('normalizes literal backslash-n and renders headings and bold', () => {
    const out = renderLegalBody('# 제목\\n## 부제\\n본문 **강조**', 'EMPTY');
    expect(out).toContain('<h2 class="text-lg font-black text-brand-400 mt-6 mb-4">제목</h2>');
    expect(out).toContain('<strong class="font-bold text-white">강조</strong>');
  });

  it('escapes stored markup so it can never execute', () => {
    const out = renderLegalBody('<script>alert(1)</script>', 'EMPTY');
    expect(out).not.toContain('<script>');
    expect(out).toContain('&lt;script&gt;');
  });
});
