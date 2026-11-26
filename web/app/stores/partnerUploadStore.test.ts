import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { partnerApi } from '@/api/endpoints';
import { usePartnerUploadStore } from '@/stores/partnerUploadStore';

vi.mock('@/api/endpoints', () => ({
  partnerApi: {
    finalize: vi.fn(),
  },
}));

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

  it('gates the R9 4-step flow (file -> precheck -> metadata -> review)', () => {
    const s = usePartnerUploadStore();
    expect(s.canLeaveStep1).toBe(false);
    s.file = new File(['x'], 'a.wav');
    expect(s.canLeaveStep1).toBe(true);
    // Step 2 exit needs a passed pre-check.
    expect(s.step2Blocked).toBe(true);
    s.gate12 = 'pass';
    s.grant = { eligible: true, stagingKey: 'k', uploadUrl: 'u', expiresInSecs: 1 };
    expect(s.step2Blocked).toBe(false);
    // Step 3 exit needs a title and no collective trust.
    expect(s.step3Blocked).toBe(true);
    s.title = 'T';
    expect(s.step3Blocked).toBe(false);
    s.collectiveTrust = 'TRUSTED';
    expect(s.step3Blocked).toBe(true);
  });
});
