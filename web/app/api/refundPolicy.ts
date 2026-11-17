// Refund policy copy lives in checkout.* locale keys (R13); this module only
// names the line order shared by pricing, checkout, and checkout-confirm.
// Mirrors backend/internal/service/billing.go (ComputeRefund) and
// RefundTermsVersion "2026-11-12-v2". Change both together.
export const REFUND_POLICY_LINE_KEYS: readonly string[] = [
  'checkout.refundL1',
  'checkout.refundL2',
  'checkout.refundL3',
  'checkout.refundL4',
  'checkout.refundL5',
  'checkout.refundL6',
];

export const REFUND_POLICY_SUMMARY_KEY = 'checkout.refundSummary';
