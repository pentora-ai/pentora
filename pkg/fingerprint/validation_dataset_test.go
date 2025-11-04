package fingerprint

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidationDataset_Smoke(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "ds.yaml")
	yaml := "true_positives:\n- protocol: http\n  port: 80\n  banner: 'Server: nginx'\n  expected_product: nginx\n  description: ok\ntrue_negatives:\n- protocol: ssh\n  port: 22\n  banner: 'HTTP/1.1 200 OK'\n  expected_match: false\n  description: tn\nedge_cases:\n- protocol: ftp\n  port: 21\n  banner: '220 FTP'\n  expected_product: vsftpd\n  description: edge\n"
	if err := os.WriteFile(p, []byte(yaml), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	ds, err := LoadValidationDataset(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ds.TruePositives) != 1 || len(ds.TrueNegatives) != 1 || len(ds.EdgeCases) != 1 {
		t.Fatalf("unexpected sizes: %+v", ds)
	}
}

func TestLoadValidationDataset_Errors(t *testing.T) {
	if _, err := LoadValidationDataset("/does/not/exist.yaml"); err == nil {
		t.Fatalf("expected read error")
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(p, []byte("not: [ yaml"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := LoadValidationDataset(p); err == nil {
		t.Fatalf("expected parse error")
	}
}
