// Analytics event-name contract (mirrors domain.AnalyticsEvents, WI-S5).
export const ANALYTICS_EVENTS = [
  'signup_started',
  'signup_completed',
  'login_succeeded',
  'play_started',
  'play_30s',
  'playlist_created',
  'track_liked',
  'inquiry_opened',
  'inquiry_submitted',
  'checkout_started',
  'checkout_completed',
  'locale_changed',
  'share_clicked',
  'embed_loaded',
  'create_waitlist_joined',
  'web_vital',
] as const;

export type AnalyticsEventName = (typeof ANALYTICS_EVENTS)[number];
