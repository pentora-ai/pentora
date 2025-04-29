package license

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"
)

// LicensePayload represents the contents of the license
type LicensePayload struct {
	Licensee     string   `json:"licensee"`
	Organization string   `json:"organization"`
	IssuedAt     string   `json:"issued_at"`
	ExpiresAt    string   `json:"expires_at"`
	Product      string   `json:"product"`
	Features     []string `json:"features"`
	LicenseType  string   `json:"license_type"`
}

// LicenseFile represents the structure of the license file
type LicenseFile struct {
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
}

// LoadPublicKey reads and parses a PEM encoded public key
func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaPubKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}
	return rsaPubKey, nil
}

// ValidateLicense validates the license file
func ValidateLicense(licensePath, pubKeyPath string) (*LicensePayload, error) {
	// Read license file
	data, err := os.ReadFile(licensePath)
	if err != nil {
		return nil, fmt.Errorf("reading license file: %w", err)
	}

	var lf LicenseFile
	if err := json.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("unmarshaling license file: %w", err)
	}

	// Decode payload and signature
	payloadBytes, err := base64.StdEncoding.DecodeString(lf.Payload)
	if err != nil {
		return nil, fmt.Errorf("decoding payload: %w", err)
	}
	signatureBytes, err := base64.StdEncoding.DecodeString(lf.Signature)
	if err != nil {
		return nil, fmt.Errorf("decoding signature: %w", err)
	}

	// Load public key
	pubKey, err := LoadPublicKey(pubKeyPath)
	if err != nil {
		return nil, fmt.Errorf("loading public key: %w", err)
	}

	// Verify signature
	hashed := sha256.Sum256(payloadBytes)
	if err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hashed[:], signatureBytes); err != nil {
		return nil, fmt.Errorf("license signature invalid: %w", err)
	}

	// Parse payload
	var payload LicensePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("unmarshaling payload: %w", err)
	}

	// Check expiration
	expTime, err := time.Parse("2006-01-02", payload.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("invalid expiration date format: %w", err)
	}
	if time.Now().After(expTime) {
		return nil, errors.New("license expired")
	}

	return &payload, nil
}

// Checks if a feature is licensed
func (lp *LicensePayload) HasFeature(feature string) bool {
	for _, f := range lp.Features {
		if f == feature {
			return true
		}
	}
	return false
}
