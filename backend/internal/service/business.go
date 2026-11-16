package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/pkg/mail"
	"avibe-platform/backend/internal/repository"
)

// BusinessService owns B2B inquiries and the generation waitlist (R7).
type BusinessService struct {
	repos       *repository.Repos
	mailer      mail.Mailer
	notifyEmail string
}

func NewBusinessService(repos *repository.Repos, mailer mail.Mailer, notifyEmail string) *BusinessService {
	return &BusinessService{repos: repos, mailer: mailer, notifyEmail: notifyEmail}
}

// InquiryInput is a validated inquiry submission (honeypot handled in handler).
type InquiryInput struct {
	Company      string
	Country      string
	StoreCount   int
	UseCase      string
	ContactName  string
	ContactEmail string
	ContactPhone string
	Message      string
}

func validEmail(email string) bool {
	email = strings.TrimSpace(email)
	if !strings.Contains(email, "@") || len(email) > 320 {
		return false
	}
	parts := strings.Split(email, "@")
	return len(parts) == 2 && parts[0] != "" && strings.Contains(parts[1], ".")
}

// SubmitInquiry validates, stores, and notifies ops (best-effort mail).
func (s *BusinessService) SubmitInquiry(ctx context.Context, in InquiryInput) (*domain.BusinessInquiry, error) {
	in.Company = strings.TrimSpace(in.Company)
	in.ContactEmail = strings.TrimSpace(in.ContactEmail)
	in.Message = strings.TrimSpace(in.Message)
	if in.Company == "" || in.Message == "" || !validEmail(in.ContactEmail) {
		return nil, fmt.Errorf("%w: company, contactEmail, and message are required", domain.ErrInvalidInput)
	}
	if len(in.Company) > 200 || len(in.Message) > 5000 {
		return nil, fmt.Errorf("%w: company/message too long", domain.ErrInvalidInput)
	}
	if in.StoreCount < 0 {
		return nil, fmt.Errorf("%w: storeCount must be >= 0", domain.ErrInvalidInput)
	}
	q, err := s.repos.Business.CreateInquiry(ctx, &domain.BusinessInquiry{
		Company:      in.Company,
		Country:      strings.TrimSpace(in.Country),
		StoreCount:   in.StoreCount,
		UseCase:      strings.TrimSpace(in.UseCase),
		ContactName:  strings.TrimSpace(in.ContactName),
		ContactEmail: in.ContactEmail,
		ContactPhone: strings.TrimSpace(in.ContactPhone),
		Message:      in.Message,
	})
	if err != nil {
		return nil, err
	}
	if s.notifyEmail != "" {
		subject := fmt.Sprintf("[AVIBE] business inquiry: %s", q.Company)
		body := fmt.Sprintf("company: %s\ncountry: %s\nstores: %d\nuse: %s\nname: %s\nemail: %s\nphone: %s\n\n%s\n",
			q.Company, q.Country, q.StoreCount, q.UseCase, q.ContactName, q.ContactEmail, q.ContactPhone, q.Message)
		if err := s.mailer.Send(ctx, s.notifyEmail, subject, body); err != nil {
			slog.Warn("business inquiry notify failed", "id", q.ID, "err", err)
		}
	}
	return q, nil
}

// ListInquiries returns newest-first inquiries (admin).
func (s *BusinessService) ListInquiries(ctx context.Context, status string, limit int) ([]*domain.BusinessInquiry, error) {
	switch status {
	case "", "NEW", "CONTACTED", "CLOSED":
		return s.repos.Business.ListInquiries(ctx, status, limit)
	default:
		return nil, fmt.Errorf("%w: status must be NEW|CONTACTED|CLOSED", domain.ErrInvalidInput)
	}
}

// SetInquiryStatus moves one inquiry (admin).
func (s *BusinessService) SetInquiryStatus(ctx context.Context, id, status string) (*domain.BusinessInquiry, error) {
	switch status {
	case "NEW", "CONTACTED", "CLOSED":
		return s.repos.Business.SetInquiryStatus(ctx, id, status)
	default:
		return nil, fmt.Errorf("%w: status must be NEW|CONTACTED|CLOSED", domain.ErrInvalidInput)
	}
}

// BusinessProfile returns the caller's company profile (R10 console).
func (s *BusinessService) BusinessProfile(ctx context.Context, userID string) (*domain.BusinessProfile, error) {
	if userID == "" {
		return nil, domain.ErrInvalidCredentials
	}
	return s.repos.Business.GetBusinessProfileByUser(ctx, userID)
}

// ListBusinessProfiles returns company profiles by status (admin).
func (s *BusinessService) ListBusinessProfiles(ctx context.Context, status string, limit int) ([]*domain.BusinessProfile, error) {
	switch status {
	case domain.PartnerPending, domain.PartnerApproved, domain.PartnerSuspended:
		return s.repos.Business.ListBusinessProfilesByStatus(ctx, status, limit)
	default:
		return nil, fmt.Errorf("%w: status must be PENDING|APPROVED|SUSPENDED", domain.ErrInvalidInput)
	}
}

// ReviewBusiness approves or suspends a company profile (admin manual).
func (s *BusinessService) ReviewBusiness(ctx context.Context, userID string, approve bool, verifierID string) (*domain.BusinessProfile, error) {
	if userID == "" || verifierID == "" {
		return nil, domain.ErrInvalidCredentials
	}
	status := domain.PartnerSuspended
	if approve {
		status = domain.PartnerApproved
	}
	return s.repos.Business.ReviewBusinessProfile(ctx, userID, status, verifierID)
}

// JoinWaitlist registers an email (idempotent).
func (s *BusinessService) JoinWaitlist(ctx context.Context, email, interest, locale string) (*domain.WaitlistEntry, bool, error) {
	email = strings.TrimSpace(email)
	if !validEmail(email) {
		return nil, false, fmt.Errorf("%w: valid email required", domain.ErrInvalidInput)
	}
	if len(interest) > 500 || len(locale) > 16 {
		return nil, false, fmt.Errorf("%w: interest/locale too long", domain.ErrInvalidInput)
	}
	return s.repos.Business.JoinWaitlist(ctx, email, strings.TrimSpace(interest), strings.TrimSpace(locale))
}
