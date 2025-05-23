// pkg/version/version_test.go
package version

import (
	"strings"
	"testing"
	"time"
)

func TestInfo_ReturnsFormattedString(t *testing.T) {
	// vars set at build-time, here using default "dev"
	info := Info()

	if !strings.Contains(info, "Pentora") {
		t.Errorf("Expected info to contain 'Pentora', got: %s", info)
	}
	if !strings.Contains(info, Version) {
		t.Errorf("Expected info to contain version '%s'", Version)
	}
	if !strings.Contains(info, Commit) {
		t.Errorf("Expected info to contain commit '%s'", Commit)
	}
	if !strings.Contains(info, BuildDate) {
		t.Errorf("Expected info to contain build date '%s'", BuildDate)
	}
}

func TestGet_ReturnsCorrectStruct(t *testing.T) {
	v := Get()

	if v.Version != Version {
		t.Errorf("Expected version %s, got %s", Version, v.Version)
	}
	if v.Commit != Commit {
		t.Errorf("Expected commit %s, got %s", Commit, v.Commit)
	}
	if v.BuildDate != BuildDate {
		t.Errorf("Expected build date %s, got %s", BuildDate, v.BuildDate)
	}
}

func TestStartDate_IsInitialized(t *testing.T) {
	if time.Since(StartDate) > time.Minute {
		t.Errorf("StartDate is too old: %s", StartDate)
	}
}

func TestCheckNewVersion_DevSkipsCheck(t *testing.T) {
	CheckNewVersion()
}
