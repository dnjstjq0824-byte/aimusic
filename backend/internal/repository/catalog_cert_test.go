package repository

import (
	"database/sql"
	"testing"
	"time"
)

// fakeRow feeds scanTrack/scanTrackWithCreator one row of values.
type fakeRow struct {
	vals []any
	err  error
}

func (f *fakeRow) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(dest) != len(f.vals) {
		panic("fakeRow arity mismatch")
	}
	for i, d := range dest {
		switch p := d.(type) {
		case *string:
			*p = f.vals[i].(string)
		case *sql.NullString:
			*p = f.vals[i].(sql.NullString)
		case *bool:
			*p = f.vals[i].(bool)
		case *sql.NullInt64:
			*p = f.vals[i].(sql.NullInt64)
		case *sql.NullFloat64:
			*p = f.vals[i].(sql.NullFloat64)
		case *int64:
			*p = f.vals[i].(int64)
		case *time.Time:
			*p = f.vals[i].(time.Time)
		default:
			panic("fakeRow unhandled type")
		}
	}
	return nil
}

func ns(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

// trackVals builds trackColumns-ordered values (34 fields + cert).
func trackVals(cert sql.NullString) []any {
	now := time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC)
	return []any{
		"t1", ns(""), "p1", "Neon", "AbC123", "neon",
		ns("electronic"), "focus", "FULL_AI", "", ns(""), ns(""), true, false, ns(""),
		sql.NullInt64{}, "staging/x.wav", ns(""), ns(""), ns(""),
		"SUNO", "EXTERNAL_AI", true, ns(""), sql.NullInt64{}, ns(""), sql.NullFloat64{}, ns(""), "READY",
		ns(""), int64(7), now, now, cert,
	}
}

func TestScanTrackCertNumber(t *testing.T) {
	with, err := scanTrack(&fakeRow{vals: trackVals(ns("AVB-TEST1234"))})
	if err != nil {
		t.Fatalf("scanTrack: %v", err)
	}
	if with.CertNumber == nil || *with.CertNumber != "AVB-TEST1234" {
		t.Errorf("CertNumber = %+v", with.CertNumber)
	}
	without, err := scanTrack(&fakeRow{vals: trackVals(sql.NullString{})})
	if err != nil {
		t.Fatalf("scanTrack: %v", err)
	}
	if without.CertNumber != nil {
		t.Errorf("CertNumber without cert = %+v, want nil", without.CertNumber)
	}
}

func TestScanTrackWithCreatorCertNumber(t *testing.T) {
	// WithCreator order is base(33) + creator + handle + cert.
	vals := append(trackVals(sql.NullString{})[:33], ns("Creator"), ns("handle1"), ns("AVB-W1"))
	got, err := scanTrackWithCreator(&fakeRow{vals: vals})
	if err != nil {
		t.Fatalf("scanTrackWithCreator: %v", err)
	}
	if got.CertNumber == nil || *got.CertNumber != "AVB-W1" {
		t.Errorf("CertNumber = %+v", got.CertNumber)
	}
	if got.CreatorName != "Creator" {
		t.Errorf("CreatorName = %q", got.CreatorName)
	}
	vals2 := append(trackVals(sql.NullString{})[:33], ns("Creator"), ns("handle1"), sql.NullString{})
	got2, err := scanTrackWithCreator(&fakeRow{vals: vals2})
	if err != nil {
		t.Fatalf("scanTrackWithCreator: %v", err)
	}
	if got2.CertNumber != nil {
		t.Errorf("CertNumber without cert = %+v, want nil", got2.CertNumber)
	}
}

func TestScanTrackBaseHasNoCert(t *testing.T) {
	got, err := scanTrackBase(&fakeRow{vals: trackVals(sql.NullString{})[:33]})
	if err != nil {
		t.Fatalf("scanTrackBase: %v", err)
	}
	if got.CertNumber != nil {
		t.Errorf("fresh track CertNumber = %+v, want nil", got.CertNumber)
	}
	if got.Title != "Neon" {
		t.Errorf("Title = %q", got.Title)
	}
}
