package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"avibe-platform/backend/internal/domain"
)

type fakeStoreRepo struct {
	businesses map[string]*domain.StoreBusiness
	devices    map[string]*domain.StoreDevice
	schedules  map[string]*domain.StoreSchedule
	logs       []*domain.StorePlayLog
	now        time.Time
}

func newFakeStoreRepo() *fakeStoreRepo {
	return &fakeStoreRepo{
		businesses: map[string]*domain.StoreBusiness{},
		devices:    map[string]*domain.StoreDevice{},
		schedules:  map[string]*domain.StoreSchedule{},
		now:        time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC), // Tuesday
	}
}

func (f *fakeStoreRepo) CreateBusiness(_ context.Context, partnerID, businessNumber, storeName string, licenseCount int) (*domain.StoreBusiness, error) {
	for _, b := range f.businesses {
		if b.BusinessNumber == businessNumber {
			return nil, errors.New("duplicate business_number")
		}
	}
	b := &domain.StoreBusiness{ID: "biz-" + businessNumber, PartnerID: partnerID, BusinessNumber: businessNumber,
		StoreName: storeName, LicenseCount: licenseCount, Status: domain.StorePending}
	f.businesses[b.ID] = b
	return b, nil
}

func (f *fakeStoreRepo) GetBusinessByID(_ context.Context, id string) (*domain.StoreBusiness, error) {
	b, ok := f.businesses[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return b, nil
}

func (f *fakeStoreRepo) ListBusinessesByPartner(_ context.Context, partnerID string) ([]*domain.StoreBusiness, error) {
	out := []*domain.StoreBusiness{}
	for _, b := range f.businesses {
		if b.PartnerID == partnerID {
			out = append(out, b)
		}
	}
	return out, nil
}

func (f *fakeStoreRepo) VerifyBusiness(_ context.Context, id, status, verifierID string) error {
	b, ok := f.businesses[id]
	if !ok {
		return domain.ErrNotFound
	}
	b.Status = status
	b.VerifiedBy = &verifierID
	return nil
}

func (f *fakeStoreRepo) CreateDevice(_ context.Context, businessID, deviceName, pairingCode string, pairingExpiry time.Time) (*domain.StoreDevice, error) {
	d := &domain.StoreDevice{ID: "dev-" + pairingCode, BusinessID: businessID, DeviceName: deviceName,
		PairingCode: &pairingCode, PairingExpiresAt: &pairingExpiry, Status: domain.DevicePending, Volume: 70}
	f.devices[d.ID] = d
	return d, nil
}

func (f *fakeStoreRepo) GetDeviceByID(_ context.Context, id string) (*domain.StoreDevice, error) {
	d, ok := f.devices[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return d, nil
}

func (f *fakeStoreRepo) GetDeviceByPairingCode(_ context.Context, code string) (*domain.StoreDevice, error) {
	for _, d := range f.devices {
		if d.Status == domain.DevicePending && d.PairingCode != nil && *d.PairingCode == code &&
			d.PairingExpiresAt != nil && d.PairingExpiresAt.After(f.now) {
			return d, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeStoreRepo) GetDeviceByTokenHash(_ context.Context, hash string) (*domain.StoreDevice, error) {
	for _, d := range f.devices {
		if d.Status == domain.DeviceActive && d.DeviceTokenHash != nil && *d.DeviceTokenHash == hash {
			return d, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeStoreRepo) ListDevicesByBusiness(_ context.Context, businessID string) ([]*domain.StoreDevice, error) {
	out := []*domain.StoreDevice{}
	for _, d := range f.devices {
		if d.BusinessID == businessID {
			out = append(out, d)
		}
	}
	return out, nil
}

func (f *fakeStoreRepo) ActivateDevice(_ context.Context, id, tokenHash string) error {
	d, ok := f.devices[id]
	if !ok {
		return domain.ErrNotFound
	}
	d.Status = domain.DeviceActive
	d.DeviceTokenHash = &tokenHash
	d.PairingCode = nil
	d.PairingExpiresAt = nil
	return nil
}

func (f *fakeStoreRepo) UpdateDevice(_ context.Context, id, deviceName string, volume int) error {
	d, ok := f.devices[id]
	if !ok {
		return domain.ErrNotFound
	}
	d.DeviceName = deviceName
	d.Volume = volume
	return nil
}

func (f *fakeStoreRepo) RevokeDevice(_ context.Context, id string) error {
	d, ok := f.devices[id]
	if !ok {
		return domain.ErrNotFound
	}
	d.Status = domain.DeviceRevoked
	d.DeviceTokenHash = nil
	return nil
}

func (f *fakeStoreRepo) TouchDeviceSeen(_ context.Context, id string) error {
	if _, ok := f.devices[id]; !ok {
		return domain.ErrNotFound
	}
	return nil
}

func (f *fakeStoreRepo) UpsertSchedule(_ context.Context, businessID string, weekday int, open, closeTime string, playlistID *string, isClosed bool) (*domain.StoreSchedule, error) {
	s := &domain.StoreSchedule{ID: "sch", BusinessID: businessID, Weekday: weekday,
		OpenTime: open, CloseTime: closeTime, PlaylistID: playlistID, IsClosed: isClosed}
	f.schedules[businessID] = s
	return s, nil
}

func (f *fakeStoreRepo) ListSchedulesByBusiness(_ context.Context, businessID string) ([]*domain.StoreSchedule, error) {
	if s, ok := f.schedules[businessID]; ok {
		return []*domain.StoreSchedule{s}, nil
	}
	return []*domain.StoreSchedule{}, nil
}

func (f *fakeStoreRepo) AppendPlayLog(_ context.Context, deviceID, trackID string, durationSecs int, counted bool) (*domain.StorePlayLog, error) {
	l := &domain.StorePlayLog{ID: "log", DeviceID: deviceID, TrackID: trackID,
		DurationPlayedSecs: durationSecs, CountedAsPlay: counted}
	f.logs = append(f.logs, l)
	return l, nil
}

func (f *fakeStoreRepo) ListPlayLogsByDevice(_ context.Context, deviceID string, limit int) ([]*domain.StorePlayLog, error) {
	out := []*domain.StorePlayLog{}
	for _, l := range f.logs {
		if l.DeviceID == deviceID {
			out = append(out, l)
		}
	}
	return out, nil
}

type fakeStoreTracks struct{ m map[string]*domain.Track }

func (f fakeStoreTracks) GetByID(_ context.Context, id string) (*domain.Track, error) {
	t, ok := f.m[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return t, nil
}

type fakeStorePartners struct{ m map[string]*domain.PartnerProfile }

func (f fakeStorePartners) GetByID(_ context.Context, id string) (*domain.PartnerProfile, error) {
	p, ok := f.m[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

type fakeStorePlaylists struct {
	playlists map[string]*domain.Playlist
	tracks    map[string][]*domain.PlaylistTrack
}

func (f fakeStorePlaylists) GetByID(_ context.Context, id string) (*domain.Playlist, error) {
	p, ok := f.playlists[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

func (f fakeStorePlaylists) ListTracks(_ context.Context, playlistID string) ([]*domain.PlaylistTrack, error) {
	return f.tracks[playlistID], nil
}

func (f fakeStorePlaylists) ListOfficialCuration(_ context.Context) ([]*domain.Playlist, error) {
	out := []*domain.Playlist{}
	for _, p := range f.playlists {
		if p.IsOfficialCuration {
			out = append(out, p)
		}
	}
	return out, nil
}

func testStoreService(f *fakeStoreRepo, tracks map[string]*domain.Track, partners map[string]*domain.PartnerProfile, playlists *fakeStorePlaylists) *StoreService {
	if playlists == nil {
		playlists = &fakeStorePlaylists{}
	}
	return &StoreService{
		store: f, tracks: fakeStoreTracks{tracks}, partners: fakeStorePartners{partners},
		playlists: playlists, now: func() time.Time { return f.now },
	}
}

func TestRegisterBusinessValidatesChecksum(t *testing.T) {
	f := newFakeStoreRepo()
	svc := testStoreService(f, nil, nil, nil)
	if _, err := svc.RegisterBusiness(context.Background(), "partner-1", "123-45-67890", "Store", 1); err == nil {
		t.Error("bad checksum must fail")
	}
	b, err := svc.RegisterBusiness(context.Background(), "partner-1", "123-45-67891", "Gangnam Store", 2)
	if err != nil {
		t.Fatalf("RegisterBusiness: %v", err)
	}
	if b.BusinessNumber != "1234567891" || b.Status != domain.StorePending || b.LicenseCount != 2 {
		t.Errorf("business = %+v", b)
	}
}

func TestClaimDeviceFullCycle(t *testing.T) {
	f := newFakeStoreRepo()
	f.businesses["biz-1"] = &domain.StoreBusiness{ID: "biz-1", PartnerID: "partner-1",
		BusinessNumber: "1234567891", StoreName: "S", LicenseCount: 1, Status: domain.StoreApproved}
	svc := testStoreService(f, nil, nil, nil)
	dev, err := svc.CreateDevice(context.Background(), "partner-1", "biz-1", "Counter Tablet")
	if err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	if dev.PairingCode == nil || len(*dev.PairingCode) != 8 {
		t.Fatalf("pairing code = %+v", dev)
	}
	code := *dev.PairingCode
	claimed, token, err := svc.ClaimDevice(context.Background(), code)
	if err != nil {
		t.Fatalf("ClaimDevice: %v", err)
	}
	if claimed.Status != domain.DeviceActive || token == "" {
		t.Errorf("claimed = %+v token=%q", claimed, token)
	}
	if _, _, err := svc.ClaimDevice(context.Background(), code); err == nil {
		t.Error("reused pairing code must fail")
	}
	// License cap: second device exceeds license_count=1.
	if _, err := svc.CreateDevice(context.Background(), "partner-1", "biz-1", "Second"); err == nil {
		t.Error("device over license cap must fail")
	}
}

func TestDeviceQueueAndHeartbeat(t *testing.T) {
	f := newFakeStoreRepo() // Tuesday 12:00 UTC
	f.businesses["biz-1"] = &domain.StoreBusiness{ID: "biz-1", PartnerID: "partner-1",
		BusinessNumber: "1234567891", StoreName: "S", LicenseCount: 2, Status: domain.StoreApproved}
	f.devices["dev-1"] = &domain.StoreDevice{ID: "dev-1", BusinessID: "biz-1",
		DeviceName: "D", Status: domain.DeviceActive, Volume: 70}
	plID := "pl-1"
	f.schedules["biz-1"] = &domain.StoreSchedule{ID: "sch", BusinessID: "biz-1",
		Weekday: int(f.now.Weekday()), OpenTime: "09:00", CloseTime: "22:00",
		PlaylistID: &plID}
	tracks := map[string]*domain.Track{
		"t1": {ID: "t1", PartnerID: "partner-1", Title: "A", Status: domain.TrackReady, HLSPrefix: strRef("hls/t1")},
		"t2": {ID: "t2", PartnerID: "partner-1", Title: "B", Status: domain.TrackTakenDown},
	}
	pl := &fakeStorePlaylists{
		playlists: map[string]*domain.Playlist{"pl-1": {ID: "pl-1", OwnerUserID: "user-1", Title: "Day"}},
		tracks: map[string][]*domain.PlaylistTrack{"pl-1": {
			{ID: "pt1", PlaylistID: "pl-1", TrackID: "t1", Position: 1},
			{ID: "pt2", PlaylistID: "pl-1", TrackID: "t2", Position: 2},
		}},
	}
	svc := testStoreService(f, tracks, nil, pl)
	q, err := svc.DeviceQueue(context.Background(), "dev-1")
	if err != nil {
		t.Fatalf("DeviceQueue: %v", err)
	}
	if len(q.Tracks) != 1 || q.Tracks[0].TrackID != "t1" {
		t.Errorf("queue = %+v", q)
	}
	counted, err := svc.Heartbeat(context.Background(), "dev-1", "t1", 45)
	if err != nil || !counted {
		t.Errorf("heartbeat counted=%v err=%v", counted, err)
	}
	counted, err = svc.Heartbeat(context.Background(), "dev-1", "t1", 10)
	if err != nil || counted {
		t.Errorf("short heartbeat counted=%v err=%v", counted, err)
	}
	if len(f.logs) != 2 {
		t.Errorf("logs = %d", len(f.logs))
	}
}

func TestScheduleValidation(t *testing.T) {
	f := newFakeStoreRepo()
	f.businesses["biz-1"] = &domain.StoreBusiness{ID: "biz-1", PartnerID: "partner-1",
		BusinessNumber: "1234567891", StoreName: "S", LicenseCount: 1, Status: domain.StoreApproved}
	pl := &fakeStorePlaylists{playlists: map[string]*domain.Playlist{
		"pl-own": {ID: "pl-own", OwnerUserID: "user-1", Title: "Own"},
		"pl-pub": {ID: "pl-pub", OwnerUserID: "other", Title: "Pub", IsPublic: true},
		"pl-prv": {ID: "pl-prv", OwnerUserID: "other", Title: "Private"},
	}}
	partners := map[string]*domain.PartnerProfile{
		"partner-1": {ID: "partner-1", UserID: "user-1", CompanyName: "C", ContactEmail: "e", Status: domain.PartnerApproved},
	}
	svc := testStoreService(f, nil, partners, pl)
	if _, err := svc.UpsertSchedule(context.Background(), "partner-1", "biz-1", 2, "09:00", "22:00", strRef("pl-own"), false); err != nil {
		t.Errorf("own playlist: %v", err)
	}
	if _, err := svc.UpsertSchedule(context.Background(), "partner-1", "biz-1", 3, "09:00", "22:00", strRef("pl-pub"), false); err != nil {
		t.Errorf("public playlist: %v", err)
	}
	if _, err := svc.UpsertSchedule(context.Background(), "partner-1", "biz-1", 4, "09:00", "22:00", strRef("pl-prv"), false); err == nil {
		t.Error("foreign private playlist must fail")
	}
	if _, err := svc.UpsertSchedule(context.Background(), "partner-1", "biz-1", 7, "09:00", "22:00", nil, false); err == nil {
		t.Error("weekday 7 must fail")
	}
	if _, err := svc.UpsertSchedule(context.Background(), "partner-1", "biz-1", 1, "9am", "22:00", nil, false); err == nil {
		t.Error("bad time must fail")
	}
	if _, err := svc.UpsertSchedule(context.Background(), "other-partner", "biz-1", 1, "09:00", "22:00", nil, false); err == nil {
		t.Error("foreign business must fail")
	}
}
