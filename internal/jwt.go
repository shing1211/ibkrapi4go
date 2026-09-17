// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"
	"time"
)

type jwtClaims struct {
	Issuer   string `json:"iss"`
	Subject  string `json:"sub"`
	Audience string `json:"aud"`
	IssuedAt int64  `json:"iat"`
	Expiry   int64  `json:"exp"`
	JWTID    string `json:"jti"`
	ClientID string `json:"client_id"`
}

type JWTConfig struct {
	ClientID   string
	TokenURL   string
	PrivateKey *rsa.PrivateKey
	Expiry     time.Duration
}

func signJWT(claims jwtClaims, key *rsa.PrivateKey) (string, error) {
	header := `{"alg":"RS256","typ":"JWT"}`
	hB64 := base64.RawURLEncoding.EncodeToString([]byte(header))

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal JWT claims: %w", err)
	}
	pB64 := base64.RawURLEncoding.EncodeToString(payload)

	data := []byte(hB64 + "." + pB64)
	digest := sha256.Sum256(data)
	sig, err := rsa.SignPKCS1v15(nil, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}
	sB64 := base64.RawURLEncoding.EncodeToString(sig)

	return hB64 + "." + pB64 + "." + sB64, nil
}

func ParsePrivateKeyFromPEM(pemData []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}
	keyBytes := block.Bytes

	// Try PKCS8 first
	if pkcs8, err := x509.ParsePKCS8PrivateKey(keyBytes); err == nil {
		if rsaKey, ok := pkcs8.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
		return nil, fmt.Errorf("PKCS8 key is not RSA")
	}

	// Try PKCS1 (RFC 3447)
	if rsaKey, err := parsePKCS1Key(keyBytes); err == nil {
		return rsaKey, nil
	}

	// Try raw RSA private key DER
	if rsaKey, err := parseRSAPrivateKey(keyBytes); err == nil {
		return rsaKey, nil
	}

	return nil, fmt.Errorf("unable to parse RSA private key from PEM")
}

func parsePKCS1Key(der []byte) (*rsa.PrivateKey, error) {
	key, err := x509.ParsePKCS1PrivateKey(der)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func parseRSAPrivateKey(der []byte) (*rsa.PrivateKey, error) {
	// Try PKCS8 (unencrypted)
	if keyInterface, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		if rsaKey, ok := keyInterface.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
		return nil, fmt.Errorf("PKCS8 key is not RSA")
	}
	// Try PKCS1
	return parsePKCS1Key(der)
}

func ParseRSAPublicKeyFromPEM(pemData []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	pubKey, ok := pubInterface.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not RSA")
	}
	return pubKey, nil
}

func BuildJWTAssertion(cfg JWTConfig) (string, error) {
	now := time.Now()
	expiry := now.Add(cfg.Expiry)
	if cfg.Expiry == 0 {
		expiry = now.Add(60 * time.Second)
	}
	claims := jwtClaims{
		Issuer:   cfg.ClientID,
		Subject:  cfg.ClientID,
		Audience: cfg.TokenURL,
		IssuedAt: now.Unix(),
		Expiry:   expiry.Unix(),
		JWTID:    newJTI(),
		ClientID: cfg.ClientID,
	}
	return signJWT(claims, cfg.PrivateKey)
}

func newJTI() string {
	data := make([]byte, 16)
	// Use time-based component + pseudo-random to create a unique jti
	t := time.Now().UnixNano()
	for i := range data {
		data[i] = byte((t >> (i % 8 * 4)) & 0xff)
	}
	var b strings.Builder
	b.Grow(len(data) * 2)
	for _, d := range data {
		fmt.Fprintf(&b, "%02x", d)
	}
	return b.String()
}
