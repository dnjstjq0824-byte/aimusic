import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('@/api/endpoints', () => ({
  partnerApi: {
    finalize: vi.fn(),
  },
}));

import { partnerApi } from '@/api/endpoints';
import { usePartnerUploadStore } from '@/stores/partnerUploadStore';

const finalizeMock = vi.mocked(partnerApi.finalize);

function readyStore() {
  const s = usePartnerUploadStore();
  s.stagedKey = 'staging/p/abc/f.wav';
  s.title = 'T';
  s.provider = 'SUNO';
  s.audioSha256 = 'abc';
  s.generatedAt = '2026-11-11T00:00:00Z';
  s.declaresOwnership = true;
  s.declaresNoVoiceClone = true;
  s.declaresTrainingConsent = true;
  s.agreesCommercialTerms = true;
  return s;
}

describe('partnerUploadStore ownership declarations', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.resetAllMocks();
    finalizeMock.mockResolvedValue({ trackId: 't-1', jobId: 'j-1' });
  });

  it('toggles AI credit parts and human contributions', () => {
    const s = usePartnerUploadStore();
    s.toggleAiCreditPart('LYRICS');
    s.toggleAiCreditPart('VOCALS');
    s.toggleAiCreditPart('LYRICS');
    expect(s.aiCreditParts).toEqual(['VOCALS']);
    s.toggleHumanContribution('ARRANGEMENT');
    expect(s.humanContributionTypes).toEqual(['ARRANGEMENT']);
  });

  it('sends declarations in the finalize payload', async () => {
    const s = readyStore();
    s.toggleAiCreditPart('LYRICS');
    s.toggleAiCreditPart('INSTRUMENTAL');
    s.toggleHumanContribution('MIX_SELECTION');
    expect(await s.finalize()).toBe(true);
    expect(finalizeMock).toHaveBeenCalledTimes(1);
    const input = finalizeMock.mock.calls[0]?.[0];
    expect(input?.aiCreditParts).toEqual(['LYRICS', 'INSTRUMENTAL']);
    expect(input?.rights.humanContributionTypes).toEqual(['MIX_SELECTION']);
  });

  it('omits empty declarations and clears them on reset', async () => {
    const s = readyStore();
    expect(await s.finalize()).toBe(true);
    const input = finalizeMock.mock.calls[0]?.[0];
    expect(input?.aiCreditParts).toBeUndefined();
    expect(input?.rights.humanContributionTypes).toBeUndefined();
    s.toggleAiCreditPart('LYRICS');
    s.reset();
    expect(s.aiCreditParts).toEqual([]);
    expect(s.humanContributionTypes).toEqual([]);
  });
});
