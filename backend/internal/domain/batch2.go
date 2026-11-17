package domain

import "time"

// OAuth providers supported for social login (WI-A1). WeChat stays out:
// adapter interface only, implementation deferred (needs a China entity).
const (
	OAuthGoogle = "GOOGLE"
	OAuthKakao  = "KAKAO"
	OAuthApple  = "APPLE"
	OAuthNaver  = "NAVER"
	OAuthLine   = "LINE"
)

// OAuthProviders lists every supported social provider in canonical order.
var OAuthProviders = []string{OAuthGoogle, OAuthKakao, OAuthApple, OAuthNaver, OAuthLine}

// ValidOAuthProvider reports whether p is a supported social provider.
func ValidOAuthProvider(p string) bool {
	for _, c := range OAuthProviders {
		if p == c {
			return true
		}
	}
	return false
}

// Billing order statuses.
const (
	BillingPending   = "PENDING"
	BillingPaid      = "PAID"
	BillingFailed    = "FAILED"
	BillingCancelled = "CANCELLED"
	BillingRefunded  = "REFUNDED"
	// BillingPartiallyRefunded marks a prorated (중도해지) refund.
	BillingPartiallyRefunded = "PARTIALLY_REFUNDED"
)

// Billing subscription statuses (billing_subscriptions.status).
const (
	BillSubActive    = "ACTIVE"
	BillSubCancelled = "CANCELLED"
	BillSubExpired   = "EXPIRED"
)

// OAuthAccount mirrors oauth_accounts.
type OAuthAccount struct {
	ID             string    `json:"id"`
	UserID         string    `json:"userId"`
	Provider       string    `json:"provider"`
	ProviderUserID string    `json:"providerUserId"`
	Email          *string   `json:"email,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// BillingPlan mirrors billing_plans.
type BillingPlan struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	NameKo    string    `json:"nameKo"`
	AmountKRW int       `json:"amountKrw"`
	Interval  string    `json:"interval"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// BillingOrder mirrors billing_orders.
type BillingOrder struct {
	ID                 string     `json:"id"`
	UserID             string     `json:"userId"`
	PlanID             string     `json:"planId"`
	OrderID            string     `json:"orderId"`
	AmountKRW          int        `json:"amountKrw"`
	Status             string     `json:"status"`
	TossPaymentKey     *string    `json:"tossPaymentKey,omitempty"`
	ApprovedAt         *time.Time `json:"approvedAt,omitempty"`
	RefundConsentAt    *time.Time `json:"refundConsentAt,omitempty"`
	RefundTermsVersion *string    `json:"refundTermsVersion,omitempty"`
	RefundedAmountKRW  *int       `json:"refundedAmountKrw,omitempty"`
	RefundedAt         *time.Time `json:"refundedAt,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

// BillingSubscription mirrors billing_subscriptions.
type BillingSubscription struct {
	ID                 string    `json:"id"`
	UserID             string    `json:"userId"`
	PlanID             string    `json:"planId"`
	OrderID            *string   `json:"orderId,omitempty"`
	Status             string    `json:"status"`
	CurrentPeriodStart time.Time `json:"currentPeriodStart"`
	CurrentPeriodEnd   time.Time `json:"currentPeriodEnd"`
	// CancelRequestedAt is set by 일반해지 (END_OF_PERIOD): access continues
	// until CurrentPeriodEnd.
	CancelRequestedAt *time.Time   `json:"cancelRequestedAt,omitempty"`
	Plan              *BillingPlan `json:"plan,omitempty"`
	CreatedAt         time.Time    `json:"createdAt"`
	UpdatedAt         time.Time    `json:"updatedAt"`
}

// PasswordResetToken mirrors password_reset_tokens (hash only; the raw token
// exists solely inside the emailed link).
type PasswordResetToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// User locales (users.locale CHECK, WI-A2). Codes match the web route locales.
const (
	LocaleKo   = "ko"
	LocaleEn   = "en"
	LocaleJa   = "ja"
	LocaleZhCn = "zh-cn"
	LocaleTh   = "th"
)

// UserLocales lists every supported user locale in canonical order.
var UserLocales = []string{LocaleKo, LocaleEn, LocaleJa, LocaleZhCn, LocaleTh}

// ValidLocale reports whether l is a supported user locale.
func ValidLocale(l string) bool {
	for _, c := range UserLocales {
		if l == c {
			return true
		}
	}
	return false
}

// Email verification purposes (email_verifications.purpose, WI-A2).
const (
	// VerifyPurposeAccount verifies a registered account's email.
	VerifyPurposeAccount = "VERIFY"
	// VerifyPurposeOAuthSignup verifies a typed email for a pending social
	// signup (A1 complete-email): the account is created on confirm.
	VerifyPurposeOAuthSignup = "OAUTH"
)

// EmailVerification mirrors email_verifications (hash only; the raw token
// exists solely inside the emailed link). UserID is nil for OAUTH rows,
// which carry the A1 signup token in RefToken instead.
type EmailVerification struct {
	ID        string
	UserID    *string
	Email     string
	Purpose   string
	TokenHash string
	RefToken  *string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}
