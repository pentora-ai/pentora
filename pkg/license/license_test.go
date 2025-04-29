package license

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func generateTestKeys(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	return privKey, &privKey.PublicKey
}

func savePEMPublicKey(t *testing.T, pub *rsa.PublicKey, path string) {
	pubBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatalf("Failed to marshal pubkey: %v", err)
	}
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})
	if err := os.WriteFile(path, pemData, 0644); err != nil {
		t.Fatalf("Failed to write pubkey file: %v", err)
	}
}

func createLicenseFile(t *testing.T, payload LicensePayload, priv *rsa.PrivateKey, path string) {
	payloadBytes, _ := json.Marshal(payload)
	hash := sha256.Sum256(payloadBytes)
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, hash[:])
	if err != nil {
		t.Fatalf("Failed to sign license: %v", err)
	}

	lf := LicenseFile{
		Payload:   base64.StdEncoding.EncodeToString(payloadBytes),
		Signature: base64.StdEncoding.EncodeToString(sig),
	}

	content, _ := json.MarshalIndent(lf, "", "  ")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("Failed to write license file: %v", err)
	}
}

func TestValidateLicense_Success(t *testing.T) {
	tmp := t.TempDir()
	priv, pub := generateTestKeys(t)

	payload := LicensePayload{
		Licensee:     "Test User",
		Organization: "Layerlog",
		IssuedAt:     "2025-01-01",
		ExpiresAt:    time.Now().Add(24 * time.Hour).Format("2006-01-02"),
		Product:      "Pentora",
		Features:     []string{"scanner"},
		LicenseType:  "test",
	}

	pubPath := filepath.Join(tmp, "pub.pem")
	licPath := filepath.Join(tmp, "valid.license")
	savePEMPublicKey(t, pub, pubPath)
	createLicenseFile(t, payload, priv, licPath)

	_, err := ValidateLicense(licPath, pubPath)
	if err != nil {
		t.Fatalf("expected license to validate, got error: %v", err)
	}
}

func TestValidateLicense_Expired(t *testing.T) {
	tmp := t.TempDir()
	priv, pub := generateTestKeys(t)

	payload := LicensePayload{
		Licensee:     "Test User",
		Organization: "Layerlog",
		IssuedAt:     "2024-01-01",
		ExpiresAt:    "2024-01-02",
		Product:      "Pentora",
		Features:     []string{"scanner"},
		LicenseType:  "test",
	}

	pubPath := filepath.Join(tmp, "pub.pem")
	licPath := filepath.Join(tmp, "expired.license")
	savePEMPublicKey(t, pub, pubPath)
	createLicenseFile(t, payload, priv, licPath)

	_, err := ValidateLicense(licPath, pubPath)
	if err == nil || err.Error() != "license expired" {
		t.Fatalf("expected license expired error, got: %v", err)
	}
}

func TestValidateLicense_InvalidSignature(t *testing.T) {
	tmp := t.TempDir()
	priv1, _ := generateTestKeys(t)
	_, pub2 := generateTestKeys(t)

	payload := LicensePayload{
		Licensee:     "Test User",
		Organization: "Layerlog",
		IssuedAt:     "2025-01-01",
		ExpiresAt:    time.Now().Add(24 * time.Hour).Format("2006-01-02"),
		Product:      "Pentora",
		Features:     []string{"scanner"},
		LicenseType:  "test",
	}

	pubPath := filepath.Join(tmp, "wrong_pub.pem")
	licPath := filepath.Join(tmp, "invalid.license")
	savePEMPublicKey(t, pub2, pubPath) // intentionally wrong pubkey
	createLicenseFile(t, payload, priv1, licPath)

	_, err := ValidateLicense(licPath, pubPath)
	if err == nil {
		t.Fatalf("expected signature validation error")
	}
}

func TestHasFeature(t *testing.T) {
	lp := &LicensePayload{
		Features: []string{"scanner", "parser"},
	}

	if !lp.HasFeature("scanner") {
		t.Errorf("expected scanner to be present")
	}

	if lp.HasFeature("exporter") {
		t.Errorf("did not expect exporter to be present")
	}
}
