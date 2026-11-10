-- 000011: store business + player devices (WI-05 #19/#20/#21). One store =
-- one license. Play counts for settlement flow only from certified devices
-- via store_play_logs.
CREATE TABLE IF NOT EXISTS store_businesses (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  partner_id UUID NOT NULL REFERENCES partner_profiles(id) ON DELETE CASCADE,
  business_number TEXT NOT NULL,
  store_name TEXT NOT NULL,
  license_count INTEGER NOT NULL DEFAULT 1 CHECK (license_count >= 1),
  status TEXT NOT NULL DEFAULT 'PENDING'
    CHECK (status IN ('PENDING','APPROVED','SUSPENDED')),
  verified_by UUID REFERENCES users(id) ON DELETE SET NULL,
  verified_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT uq_store_business_number UNIQUE (business_number)
);
CREATE INDEX IF NOT EXISTS idx_store_businesses_partner ON store_businesses (partner_id);
CREATE INDEX IF NOT EXISTS idx_store_businesses_status ON store_businesses (status);

CREATE TABLE IF NOT EXISTS store_devices (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  business_id UUID NOT NULL REFERENCES store_businesses(id) ON DELETE CASCADE,
  device_name TEXT NOT NULL,
  pairing_code TEXT,
  pairing_expires_at TIMESTAMPTZ,
  device_token_hash TEXT,
  status TEXT NOT NULL DEFAULT 'PENDING'
    CHECK (status IN ('PENDING','ACTIVE','REVOKED')),
  volume INTEGER NOT NULL DEFAULT 70 CHECK (volume >= 0 AND volume <= 100),
  last_seen_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_store_devices_business ON store_devices (business_id);
CREATE INDEX IF NOT EXISTS idx_store_devices_pairing ON store_devices (pairing_code)
  WHERE status = 'PENDING' AND pairing_code IS NOT NULL;

CREATE TABLE IF NOT EXISTS store_schedules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  business_id UUID NOT NULL REFERENCES store_businesses(id) ON DELETE CASCADE,
  weekday INTEGER NOT NULL CHECK (weekday >= 0 AND weekday <= 6),
  open_time TIME NOT NULL,
  close_time TIME NOT NULL,
  playlist_id UUID REFERENCES playlists(id) ON DELETE SET NULL,
  is_closed BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT uq_store_schedule_day UNIQUE (business_id, weekday)
);
CREATE INDEX IF NOT EXISTS idx_store_schedules_business ON store_schedules (business_id);

CREATE TABLE IF NOT EXISTS store_play_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  device_id UUID NOT NULL REFERENCES store_devices(id) ON DELETE CASCADE,
  track_id UUID NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
  played_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  duration_played_secs INTEGER NOT NULL DEFAULT 0 CHECK (duration_played_secs >= 0),
  counted_as_play BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_store_play_logs_device ON store_play_logs (device_id, played_at DESC);
CREATE INDEX IF NOT EXISTS idx_store_play_logs_track ON store_play_logs (track_id, played_at DESC);
