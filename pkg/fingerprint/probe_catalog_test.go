package fingerprint

import (
	"testing"
)

func TestGetProbeCatalog(t *testing.T) {
	catalog, err := GetProbeCatalog()
	if err != nil {
		t.Fatalf("GetProbeCatalog error: %v", err)
	}
	if catalog == nil {
		t.Fatal("expected catalog, got nil")
	}

	probes := catalog.ProbesFor(80, []string{"http"})
	if len(probes) == 0 {
		t.Fatalf("expected probes for port 80 http hint")
	}

	seen := map[string]struct{}{}
	for _, p := range probes {
		seen[p.ID] = struct{}{}
	}
	if _, ok := seen["http-get"]; !ok {
		t.Fatalf("expected http-get probe in result: %v", probes)
	}
}

func TestProbeCatalogValidate(t *testing.T) {
	catalog := ProbeCatalog{
		Groups: []ProbeGroup{
			{
				ID:     "test",
				Probes: []ProbeSpec{{ID: "p", Protocol: "test", Payload: "PING"}},
			},
		},
	}
	if err := catalog.Validate(); err != nil {
		t.Fatalf("unexpected validate error: %v", err)
	}
}
