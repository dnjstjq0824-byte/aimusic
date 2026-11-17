// SEO address helpers (WI-S1/S5): slugs, short IDs, creator handles,
// sitemap types, and the cookieless analytics event allowlist.
package domain

import (
	"crypto/rand"
	"errors"
	"strings"
	"unicode"
)

// SEOLocales is the public-page locale set (ko has no URL prefix).
var SEOLocales = []string{"ko", "en", "ja", "zh-cn", "th"}

// IsSEOLocale reports whether l is a public-page locale.
func IsSEOLocale(l string) bool {
	switch strings.ToLower(strings.TrimSpace(l)) {
	case "ko", "en", "ja", "zh-cn", "th":
		return true
	}
	return false
}

const shortIDAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
const shortIDLength = 8

// NewShortID returns a random base62 short ID (8 chars).
func NewShortID() (string, error) {
	var raw [6]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	var bits uint64
	for _, b := range raw {
		bits = bits<<8 | uint64(b)
	}
	out := make([]byte, shortIDLength)
	for i := shortIDLength - 1; i >= 0; i-- {
		out[i] = shortIDAlphabet[bits%62]
		bits /= 62
	}
	return string(out), nil
}

// ValidateShortID reports whether s is a base62 8-char short ID.
func ValidateShortID(s string) bool {
	if len(s) != shortIDLength {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z') {
			return false
		}
	}
	return true
}

// Slugify lowercases a title and turns whitespace into dashes (unicode kept).
// URL delimiters become dashes, runs collapse, output caps at 80 runes.
// Empty input yields fallback ("track"/"playlist").
func Slugify(title, fallback string) string {
	lowered := strings.ToLower(strings.TrimSpace(title))
	var b strings.Builder
	b.Grow(len(lowered))
	prevDash := true
	for _, r := range lowered {
		if unicode.IsSpace(r) || r == '/' || r == '\\' || r == '?' || r == '#' || r == '%' || r == '&' || r == '+' || r == '=' {
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
			continue
		}
		if unicode.IsControl(r) {
			continue
		}
		prevDash = false
		b.WriteRune(r)
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = fallback
	}
	if n := len([]rune(out)); n > 80 {
		runes := []rune(out)
		out = strings.Trim(string(runes[:80]), "-")
		if out == "" {
			out = fallback
		}
	}
	return out
}

// ReservedHandles blocks route collisions and impersonation on /creators/:handle.
var ReservedHandles = map[string]bool{
	"admin": true, "administrator": true, "api": true, "app": true,
	"tracks": true, "playlists": true, "creators": true, "explore": true,
	"charts": true, "guides": true, "guide": true, "embed": true,
	"login": true, "signup": true, "logout": true, "oauth": true,
	"account": true, "library": true, "checkout": true, "pricing": true,
	"business": true, "studio": true, "partner": true, "partners": true,
	"admin_": true, "support": true, "help": true, "trust": true,
	"legal": true, "certificate": true, "certificates": true, "store": true,
	"create": true, "for-creators": true, "for_creators": true, "avibe": true,
	"official": true, "verified": true, "system": true, "null": true, "undefined": true,
	"me": true, "settings": true, "static": true, "assets": true, "robots": true,
	"sitemap": true, "oembed": true, "events": true, "seo": true, "healthz": true, "readyz": true,
}

// ValidateHandle checks the creator handle rule: lowercase alnum + underscore,
// 3-30 chars, no reserved word.
func ValidateHandle(h string) error {
	if len(h) < 3 || len(h) > 30 {
		return errors.New("handle must be 3-30 chars")
	}
	for i := 0; i < len(h); i++ {
		c := h[i]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_') {
			return errors.New("handle must be lowercase alnum or underscore")
		}
	}
	if ReservedHandles[h] {
		return errors.New("handle is reserved")
	}
	return nil
}

// Slug param kinds for /tracks/:param and /playlists/:param.
const (
	SlugKindInvalid   = "invalid"
	SlugKindUUID      = "uuid"
	SlugKindShortID   = "short"
	SlugKindSlugShort = "slugshort"
)

// SplitSlugID classifies a detail-page param: uuid, bare short ID, or
// slug-shortID (split on the LAST dash so unicode slugs with dashes work).
func SplitSlugID(param string) (slug, shortID, kind string) {
	p := strings.TrimSpace(param)
	if p == "" {
		return "", "", SlugKindInvalid
	}
	if IsUUIDLike(p) {
		return "", "", SlugKindUUID
	}
	if ValidateShortID(p) {
		return "", p, SlugKindShortID
	}
	if i := strings.LastIndex(p, "-"); i > 0 && i+1 < len(p) {
		tail := p[i+1:]
		if ValidateShortID(tail) {
			return p[:i], tail, SlugKindSlugShort
		}
	}
	return "", "", SlugKindInvalid
}

// IsUUIDLike reports whether s looks like a uuid (36 chars, 4 dashes).
func IsUUIDLike(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := 0; i < 36; i++ {
		c := s[i]
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
				return false
			}
		}
	}
	return true
}

// CanonicalTrackPath builds /tracks/{slug}-{shortID}.
func CanonicalTrackPath(slug, shortID string) string {
	return "/tracks/" + slug + "-" + shortID
}

// CanonicalPlaylistPath builds /playlists/{slug}-{shortID}.
func CanonicalPlaylistPath(slug, shortID string) string {
	return "/playlists/" + slug + "-" + shortID
}

// Sitemap types served by GET /api/v1/seo/sitemap/{type}.
const (
	SitemapTracks    = "tracks"
	SitemapPlaylists = "playlists"
	SitemapCreators  = "creators"
	SitemapLandings  = "landings"
	SitemapGuides    = "guides"
	SitemapStatic    = "static"
)

// ValidSitemapType reports whether t is a known sitemap type.
func ValidSitemapType(t string) bool {
	switch t {
	case SitemapTracks, SitemapPlaylists, SitemapCreators, SitemapLandings, SitemapGuides, SitemapStatic:
		return true
	}
	return false
}

// AnalyticsEvents is the product-analysis allowlist (product spec 1.4, WI-S5).
// Collection is cookieless and carries no personal identifiers.
var AnalyticsEvents = map[string]bool{
	"signup_started": true, "signup_completed": true, "login_succeeded": true,
	"play_started": true, "play_30s": true, "playlist_created": true,
	"track_liked": true, "inquiry_opened": true, "inquiry_submitted": true,
	"checkout_started": true, "checkout_completed": true, "locale_changed": true,
	"share_clicked": true, "embed_loaded": true, "create_waitlist_joined": true,
	// Web-vitals RUM samples ride the same endpoint (S5).
	"web_vital": true,
}

// ValidAnalyticsEvent reports whether name may be collected.
func ValidAnalyticsEvent(name string) bool {
	return AnalyticsEvents[name]
}

// ForbiddenEventProps are never stored (PII / secret leak guard).
var ForbiddenEventProps = map[string]bool{
	"email": true, "phone": true, "name": true, "ip": true, "ip_address": true,
	"token": true, "access_token": true, "refresh_token": true, "password": true,
	"cookie": true, "cookies": true, "authorization": true, "user_agent_full": true,
	"address": true, "account": true, "card": true,
}

// SitemapItem is one row of GET /api/v1/seo/sitemap/{type}.
type SitemapItem struct {
	// Loc is the path suffix without locale prefix (e.g. /tracks/slug-abc123XY).
	Loc     string `json:"loc"`
	Lastmod string `json:"lastmod"`
}

// CreatorProfile is the public /creators/{handle} payload (S4).
type CreatorProfile struct {
	Handle      string `json:"handle"`
	DisplayName string `json:"displayName"`
	UserID      string `json:"userId"`
	PartnerID   string `json:"partnerId"`
	TrackCount  int    `json:"trackCount"`
	TotalPlays  int64  `json:"totalPlays"`
}

// AnalyticsEvent is one validated cookieless product event (S5).
type AnalyticsEvent struct {
	Name        string            `json:"name"`
	Locale      string            `json:"locale,omitempty"`
	Path        string            `json:"path,omitempty"`
	SessionHash string            `json:"sessionHash,omitempty"`
	Props       map[string]string `json:"props,omitempty"`
}

// SanitizeEventProps drops forbidden keys and caps prop count/size.
func SanitizeEventProps(props map[string]string) map[string]string {
	if len(props) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(props))
	for k, v := range props {
		if len(out) >= 20 {
			break
		}
		kl := strings.ToLower(strings.TrimSpace(k))
		if kl == "" || ForbiddenEventProps[kl] {
			continue
		}
		if len(k) > 64 {
			continue
		}
		if len(v) > 256 {
			v = v[:256]
		}
		out[k] = v
	}
	return out
}
