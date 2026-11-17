package service

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	appleIssuer   = "https://appleid.apple.com"
	lineIssuer    = "https://access.line.me"
	oidcClockSkew = 60 * time.Second
	jwksCacheTTL  = time.Hour
)

type jwksEntry struct {
	keys    map[string]crypto.PublicKey
	fetched time.Time
}

// jwksKey returns the cached public key for kid, refetching when stale.
func (s *OAuthService) jwksKey(ctx context.Context, jwksURL, kid string) (crypto.PublicKey, error) {
	s.jwksMu.Lock()
	defer s.jwksMu.Unlock()
	if e, ok := s.jwks[jwksURL]; ok && s.clock().Sub(e.fetched) < jwksCacheTTL {
		if k, ok := e.keys[kid]; ok {
			return k, nil
		}
	}
	keys, err := fetchJWKS(ctx, s.client, jwksURL)
	if err != nil {
		return nil, err
	}
	if s.jwks == nil {
		s.jwks = map[string]*jwksEntry{}
	}
	s.jwks[jwksURL] = &jwksEntry{keys: keys, fetched: s.clock()}
	k, ok := keys[kid]
	if !ok {
		return nil, fmt.Errorf("jwks: unknown kid %q", kid)
	}
	return k, nil
}

type jwksDoc struct {
	Keys []struct {
		Kty string `json:"kty"`
		Kid string `json:"kid"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

// fetchJWKS loads the provider signing keys (Apple and LINE both sign
// id_tokens with RS256).
func fetchJWKS(ctx context.Context, client *http.Client, jwksURL string) (map[string]crypto.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("jwks fetch failed (http %d)", res.StatusCode)
	}
	var doc jwksDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("jwks fetch failed: bad payload")
	}
	out := map[string]crypto.PublicKey{}
	for _, k := range doc.Keys {
		if k.Kty != "RSA" || k.Kid == "" {
			continue
		}
		nb, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			continue
		}
		eb, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			continue
		}
		e := 0
		for _, b := range eb {
			e = e<<8 | int(b)
		}
		if e == 0 {
			continue
		}
		out[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(nb), E: e}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("jwks: no RSA keys")
	}
	return out, nil
}

// verifyIDToken checks an OIDC id_token: RS256 signature via the provider
// JWKS, then iss/aud/exp/nonce. Claims use the service clock (testable).
func (s *OAuthService) verifyIDToken(ctx context.Context, jwksURL, raw, wantIss, wantAud, wantNonce string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("id_token missing")
	}
	unverified, _, err := new(jwt.Parser).ParseUnverified(raw, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("id_token parse: %w", err)
	}
	if _, ok := unverified.Method.(*jwt.SigningMethodRSA); !ok {
		return nil, fmt.Errorf("id_token alg %q rejected (RS256 only)", unverified.Method.Alg())
	}
	kid, _ := unverified.Header["kid"].(string)
	if kid == "" {
		return nil, fmt.Errorf("id_token kid missing")
	}
	key, err := s.jwksKey(ctx, jwksURL, kid)
	if err != nil {
		return nil, err
	}
	claims := jwt.MapClaims{}
	vt, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("id_token alg changed")
		}
		return key, nil
	}, jwt.WithoutClaimsValidation())
	if err != nil || !vt.Valid {
		return nil, fmt.Errorf("id_token signature: %w", err)
	}
	now := s.clock()
	exp, err := claims.GetExpirationTime()
	if err != nil || exp == nil || now.After(exp.Time.Add(oidcClockSkew)) {
		return nil, fmt.Errorf("id_token expired")
	}
	if iss, err := claims.GetIssuer(); err != nil || iss != wantIss {
		return nil, fmt.Errorf("id_token iss mismatch")
	}
	aud, err := claims.GetAudience()
	if err != nil || !audContains(aud, wantAud) {
		return nil, fmt.Errorf("id_token aud mismatch")
	}
	nonce, _ := claims["nonce"].(string)
	if nonce == "" || subtle.ConstantTimeCompare([]byte(nonce), []byte(wantNonce)) != 1 {
		return nil, fmt.Errorf("id_token nonce mismatch")
	}
	out := map[string]any{}
	for k, v := range claims {
		out[k] = v
	}
	return out, nil
}

func audContains(aud jwt.ClaimStrings, want string) bool {
	for _, a := range aud {
		if a == want {
			return true
		}
	}
	return false
}

func claimString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// claimBoolish reads a boolean claim that providers may encode as bool or
// string (Apple sends email_verified as "true").
func claimBoolish(m map[string]any, key string) bool {
	switch v := m[key].(type) {
	case bool:
		return v
	case string:
		return v == "true" || v == "1"
	default:
		return false
	}
}

// appleClientSecret mints the ES256 client_secret JWT per token request
// (short 5-minute expiry) from the p8 private key.
func appleClientSecret(teamID, keyID, servicesID, privatePEM string, now time.Time) (string, error) {
	block, _ := pem.Decode([]byte(privatePEM))
	if block == nil {
		return "", fmt.Errorf("apple private key: bad PEM")
	}
	raw, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("apple private key: %w", err)
	}
	ec, ok := raw.(*ecdsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("apple private key: want ECDSA P-256")
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"iss": teamID,
		"iat": now.Unix(),
		"exp": now.Add(5 * time.Minute).Unix(),
		"aud": appleIssuer,
		"sub": servicesID,
	})
	tok.Header["kid"] = keyID
	return tok.SignedString(ec)
}
