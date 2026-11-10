package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"regexp"

	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/repository"
)

var certNumberPattern = regexp.MustCompile(`^AVB-[A-Z2-9]{10}$`)

// certAlphabet excludes ambiguous 0/O/1/I.
const certAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// GenerateCertNumber mints an unguessable public certificate number.
func GenerateCertNumber() (string, error) {
	b := make([]byte, 10)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(certAlphabet))))
		if err != nil {
			return "", err
		}
		b[i] = certAlphabet[n.Int64()]
	}
	return "AVB-" + string(b), nil
}

// ValidCertNumber reports whether s is a well-formed certificate number.
func ValidCertNumber(s string) bool { return certNumberPattern.MatchString(s) }

// CertificateService issues and verifies per-track certificates (P2-6 #24).
type CertificateService struct {
	repos *repository.Repos
}

// NewCertificateService builds the certificate service.
func NewCertificateService(repos *repository.Repos) *CertificateService {
	return &CertificateService{repos: repos}
}

// CertificateView is the public verification payload: the credential plus
// the evidence it attests (provider plan, generation time, gate passes).
type CertificateView struct {
	Certificate *domain.TrackCertificate `json:"certificate"`
	TrackID     string                   `json:"trackId"`
	Title       string                   `json:"title"`
	AIProvider  string                   `json:"aiProvider"`
	AIModel     *string                  `json:"aiModel,omitempty"`
	ISRC        *string                  `json:"isrc,omitempty"`
	Status      string                   `json:"status"`
	PlanTier    *string                  `json:"planTier,omitempty"`
	GeneratedAt *string                  `json:"generatedAt,omitempty"`
	GatesPassed bool                     `json:"gatesPassed"`
}

// Issue creates a track's certificate (idempotent per track). Only READY
// tracks certify; issuance is an explicit admin act, not automatic.
func (s *CertificateService) Issue(ctx context.Context, trackID, actorID string) (*domain.TrackCertificate, error) {
	if trackID == "" {
		return nil, fmt.Errorf("%w: track", domain.ErrInvalidInput)
	}
	track, err := s.repos.Tracks.GetByID(ctx, trackID)
	if err != nil {
		return nil, err
	}
	if track.Status != domain.TrackReady {
		return nil, fmt.Errorf("%w: only READY tracks certify", domain.ErrInvalidInput)
	}
	if existing, err := s.repos.Certificates.GetByTrack(ctx, trackID); err == nil {
		return existing, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	number, err := GenerateCertNumber()
	if err != nil {
		return nil, err
	}
	return s.repos.Certificates.Create(ctx, trackID, number, actorID)
}

// Verify resolves a public certificate number to its attested view.
// Unknown or malformed numbers are ErrNotFound (no oracle for scanners).
func (s *CertificateService) Verify(ctx context.Context, number string) (*CertificateView, error) {
	if !ValidCertNumber(number) {
		return nil, domain.ErrNotFound
	}
	cert, err := s.repos.Certificates.GetByNumber(ctx, number)
	if err != nil {
		return nil, err
	}
	return s.view(ctx, cert)
}

// VerifyByTrack resolves a track's certificate (ErrNotFound when none).
func (s *CertificateService) VerifyByTrack(ctx context.Context, trackID string) (*CertificateView, error) {
	if trackID == "" {
		return nil, domain.ErrNotFound
	}
	cert, err := s.repos.Certificates.GetByTrack(ctx, trackID)
	if err != nil {
		return nil, err
	}
	return s.view(ctx, cert)
}

func (s *CertificateService) view(ctx context.Context, cert *domain.TrackCertificate) (*CertificateView, error) {
	track, err := s.repos.Tracks.GetByID(ctx, cert.TrackID)
	if err != nil {
		return nil, err
	}
	view := &CertificateView{
		Certificate: cert,
		TrackID:     track.ID,
		Title:       track.Title,
		AIProvider:  track.AIProvider,
		AIModel:     track.AIModelVersion,
		ISRC:        track.ISRC,
		Status:      track.Status,
	}
	if rights, err := s.repos.Rights.GetByTrack(ctx, cert.TrackID); err == nil {
		view.PlanTier = rights.PlanTierUsed
		if rights.GeneratedAt != nil {
			gen := rights.GeneratedAt.UTC().Format("2006-01-02")
			view.GeneratedAt = &gen
		}
		view.GatesPassed = rights.Gate1PolicyPassed && rights.Gate2DatePassed && rights.Gate3FFprobePassed
	}
	return view, nil
}
