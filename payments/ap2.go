package payments

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"strings"
	"time"
)

type JWK struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

type AP2Constraints struct {
	MaxAmount         float64  `json:"max_amount"`
	Currency          string   `json:"currency"`
	MerchantAllowlist []string `json:"merchant_allowlist"`
}

type AP2Subject struct {
	AgentKey    JWK            `json:"agentKey"`
	Constraints AP2Constraints `json:"constraints"`
}

type AP2MandateClaims struct {
	Issuer            string     `json:"iss"`
	Subject           string     `json:"sub"`
	IssuedAt          int64      `json:"iat"`
	Expiration        int64      `json:"exp"`
	CredentialSubject AP2Subject `json:"credentialSubject"`
	VC                *struct {
		CredentialSubject AP2Subject `json:"credentialSubject"`
	} `json:"vc,omitempty"`
}

func (c *AP2MandateClaims) GetSubject() AP2Subject {
	if c.VC != nil {
		return c.VC.CredentialSubject
	}
	return c.CredentialSubject
}

type AgentAssertionClaims struct {
	OrderID  string  `json:"order_id"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Nonce    string  `json:"nonce"`
	IssuedAt int64   `json:"iat"`
}

type AP2VerificationInput struct {
	MandateJWT     string         `json:"mandate_jwt"`     // User-signed mandate VC (JWT)
	AgentAssertion string         `json:"agent_assertion"` // Agent-signed JWT asserting the transaction
	OrderID        string         `json:"order_id"`
	Provider       string         `json:"provider"`        // "stripe", "razorpay", "mock"
	Payload        map[string]any `json:"payload"`         // Additional gateway payload (Stripe payment method token, etc.)
}

// VerifyAP2Mandate cryptographically validates the user's mandate and the agent's assertion.
func VerifyAP2Mandate(ctx context.Context, input AP2VerificationInput, amount float64, currency string, merchantHost string, userPublicKeyPEM string) (*AP2MandateClaims, error) {
	// 1. Verify User Mandate signature
	mandateParts := strings.Split(input.MandateJWT, ".")
	if len(mandateParts) != 3 {
		return nil, fmt.Errorf("invalid mandate JWT format")
	}

	userPubKey, err := parseECPublicKey(userPublicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("invalid user public key: %w", err)
	}

	signingInput := mandateParts[0] + "." + mandateParts[1]
	if err := verifyES256Signature(signingInput, mandateParts[2], userPubKey); err != nil {
		return nil, fmt.Errorf("user mandate signature verification failed: %w", err)
	}

	// Parse Mandate Claims
	payloadBytes, err := base64DecodeURL(mandateParts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode mandate payload: %w", err)
	}

	var claims AP2MandateClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse mandate claims: %w", err)
	}

	// 2. Verify Mandate Constraints (expiration, amount, currency, merchant)
	if err := claims.VerifyConstraints(amount, currency, merchantHost); err != nil {
		return nil, fmt.Errorf("mandate constraints violated: %w", err)
	}

	// 3. Verify Agent Assertion signature
	assertionParts := strings.Split(input.AgentAssertion, ".")
	if len(assertionParts) != 3 {
		return nil, fmt.Errorf("invalid agent assertion JWT format")
	}

	subj := claims.GetSubject()
	agentPubKey, err := parseJWK(subj.AgentKey)
	if err != nil {
		return nil, fmt.Errorf("invalid agent public key in mandate: %w", err)
	}

	assertionSigningInput := assertionParts[0] + "." + assertionParts[1]
	if err := verifyES256Signature(assertionSigningInput, assertionParts[2], agentPubKey); err != nil {
		return nil, fmt.Errorf("agent assertion signature verification failed: %w", err)
	}

	// Parse Agent Assertion Claims
	assertionPayloadBytes, err := base64DecodeURL(assertionParts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode agent assertion payload: %w", err)
	}

	var assertionClaims AgentAssertionClaims
	if err := json.Unmarshal(assertionPayloadBytes, &assertionClaims); err != nil {
		return nil, fmt.Errorf("failed to parse agent assertion claims: %w", err)
	}

	// Verify Assertion details match the transaction
	if assertionClaims.OrderID != input.OrderID {
		return nil, fmt.Errorf("assertion order ID mismatch: expected %s, got %s", input.OrderID, assertionClaims.OrderID)
	}
	if assertionClaims.Amount != amount {
		return nil, fmt.Errorf("assertion amount mismatch: expected %f, got %f", amount, assertionClaims.Amount)
	}
	if strings.ToUpper(assertionClaims.Currency) != strings.ToUpper(currency) {
		return nil, fmt.Errorf("assertion currency mismatch: expected %s, got %s", currency, assertionClaims.Currency)
	}

	return &claims, nil
}

func (c *AP2MandateClaims) VerifyConstraints(amount float64, currency string, merchantHost string) error {
	now := time.Now().Unix()
	if c.Expiration > 0 && now > c.Expiration {
		return fmt.Errorf("mandate has expired")
	}

	subj := c.GetSubject()

	if amount > subj.Constraints.MaxAmount {
		return fmt.Errorf("amount %f exceeds max spending limit of %f", amount, subj.Constraints.MaxAmount)
	}

	if subj.Constraints.Currency != "" && strings.ToUpper(currency) != strings.ToUpper(subj.Constraints.Currency) {
		return fmt.Errorf("currency mismatch: expected %s, got %s", subj.Constraints.Currency, currency)
	}

	if len(subj.Constraints.MerchantAllowlist) > 0 {
		allowed := false
		for _, m := range subj.Constraints.MerchantAllowlist {
			if strings.Contains(strings.ToLower(merchantHost), strings.ToLower(m)) || strings.Contains(strings.ToLower(m), strings.ToLower(merchantHost)) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("merchant %s is not in the mandate allowlist", merchantHost)
		}
	}

	return nil
}

func parseECPublicKey(pemStr string) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	ecPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an ECDSA public key")
	}
	return ecPub, nil
}

func parseJWK(jwk JWK) (*ecdsa.PublicKey, error) {
	if jwk.Kty != "EC" || jwk.Crv != "P-256" {
		return nil, fmt.Errorf("unsupported key type or curve: kty=%s, crv=%s", jwk.Kty, jwk.Crv)
	}
	xBytes, err := base64DecodeURL(jwk.X)
	if err != nil {
		return nil, fmt.Errorf("failed to decode X coordinate: %w", err)
	}
	yBytes, err := base64DecodeURL(jwk.Y)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Y coordinate: %w", err)
	}
	return &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}, nil
}

func verifyES256Signature(signingInput string, signatureB64 string, pubKey *ecdsa.PublicKey) error {
	sigBytes, err := base64DecodeURL(signatureB64)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	if len(sigBytes) != 64 {
		return fmt.Errorf("invalid signature length: expected 64 bytes for ES256, got %d", len(sigBytes))
	}

	r := new(big.Int).SetBytes(sigBytes[0:32])
	s := new(big.Int).SetBytes(sigBytes[32:64])

	hash := sha256.Sum256([]byte(signingInput))
	if !ecdsa.Verify(pubKey, hash[:], r, s) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}

func base64DecodeURL(s string) ([]byte, error) {
	// Pad string if necessary
	if l := len(s) % 4; l > 0 {
		s += strings.Repeat("=", 4-l)
	}
	// Decode base64url or fallback to base64 std
	data, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		data, err = base64.RawURLEncoding.DecodeString(s)
		if err != nil {
			data, err = base64.StdEncoding.DecodeString(s)
		}
	}
	return data, err
}
