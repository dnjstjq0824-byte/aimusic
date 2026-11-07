package domain

import "time"

// OAuth providers supported for social login.
const (
	OAuthGoogle = "GOOGLE"
	OAuthKakao  = "KAKAO"
)

// ValidOAuthProvider reports whether p is a supported social provider.
func ValidOAuthProvider(p string) bool {
	return p == OAuthGoogle || p == OAuthKakao
}

// Billing order statuses.
const (
	BillingPending   = "PENDING"
	BillingPaid      = "PAID"
	BillingFailed    = "FAILED"
	BillingCancelled = "CANCELLED"
	BillingRefunded  = "REFUNDED"
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
	ID             string     `json:"id"`
	UserID         string     `json:"userId"`
	PlanID         string     `json:"planId"`
	OrderID        string     `json:"orderId"`
	AmountKRW      int        `json:"amountKrw"`
	Status         string     `json:"status"`
	TossPaymentKey *string    `json:"tossPaymentKey,omitempty"`
	ApprovedAt     *time.Time `json:"approvedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// BillingSubscription mirrors billing_subscriptions.
type BillingSubscription struct {
	ID                 string       `json:"id"`
	UserID             string       `json:"userId"`
	PlanID             string       `json:"planId"`
	OrderID            *string      `json:"orderId,omitempty"`
	Status             string       `json:"status"`
	CurrentPeriodStart time.Time    `json:"currentPeriodStart"`
	CurrentPeriodEnd   time.Time    `json:"currentPeriodEnd"`
	Plan               *BillingPlan `json:"plan,omitempty"`
	CreatedAt          time.Time    `json:"createdAt"`
	UpdatedAt          time.Time    `json:"updatedAt"`
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
