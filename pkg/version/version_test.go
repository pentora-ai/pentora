// pkg/version/version_test.go
package version

import (
	"testing"
)

func TestVersion_String_ReturnsVersionField(t *testing.T) {
	tests := []struct {
		name    string
		version Version
		want    string
	}{
		{
			name:    "returns version string",
			version: Version{Version: "1.2.3"},
			want:    "1.2.3",
		},
		{
			name:    "returns empty string if version is empty",
			version: Version{Version: ""},
			want:    "",
		},
		{
			name:    "returns pre-release version",
			version: Version{Version: "0.1.0-beta"},
			want:    "0.1.0-beta",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.version.String()
			if got != tt.want {
				t.Errorf("Version.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetVersion(t *testing.T) {
	origVersion := version
	origCommit := commit
	origBuildDate := buildDate
	origTag := tag
	defer func() {
		version = origVersion
		commit = origCommit
		buildDate = origBuildDate
		tag = origTag
	}()

	tests := []struct {
		name      string
		setVars   func()
		want      Version
		wantStart string // expected prefix for Version.Version
	}{
		{
			name: "commit and tag set",
			setVars: func() {
				version = "1.0.0"
				commit = "abcdef1234567890"
				buildDate = "2024-06-01T12:00:00Z"
				tag = "v1.0.0"
			},
			want: Version{
				Version:   "v1.0.0",
				Commit:    "abcdef1234567890",
				BuildDate: "2024-06-01T12:00:00Z",
				Tag:       "v1.0.0",
			},
			wantStart: "v1.0.0",
		},
		{
			name: "no tag, commit >= 7 chars",
			setVars: func() {
				version = "2.1.3"
				commit = "1234567deadbeef"
				buildDate = "2024-06-02T13:00:00Z"
				tag = ""
			},
			want: Version{
				Version:   "v2.1.3+1234567",
				Commit:    "1234567deadbeef",
				BuildDate: "2024-06-02T13:00:00Z",
				Tag:       "",
			},
			wantStart: "v2.1.3+1234567",
		},
		{
			name: "no tag, commit < 7 chars",
			setVars: func() {
				version = "0.0.1"
				commit = "abc"
				buildDate = "2024-06-03T14:00:00Z"
				tag = ""
			},
			want: Version{
				Version:   "v0.0.1+unknown",
				Commit:    "abc",
				BuildDate: "2024-06-03T14:00:00Z",
				Tag:       "",
			},
			wantStart: "v0.0.1+unknown",
		},
		{
			name: "all defaults",
			setVars: func() {
				version = "dev"
				commit = ""
				buildDate = "1970-01-01T00:00:00Z"
				tag = ""
			},
			want: Version{
				Version:   "vdev+unknown",
				Commit:    "",
				BuildDate: "1970-01-01T00:00:00Z",
				Tag:       "",
			},
			wantStart: "vdev+unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setVars()
			got := GetVersion()
			if got.Version != tt.want.Version && tt.wantStart != "" && got.Version != "" {
				if len(got.Version) < len(tt.wantStart) || got.Version[:len(tt.wantStart)] != tt.wantStart {
					t.Errorf("GetVersion().Version = %q, want prefix %q", got.Version, tt.wantStart)
				}
			}
			if got.Commit != tt.want.Commit {
				t.Errorf("GetVersion().Commit = %q, want %q", got.Commit, tt.want.Commit)
			}
			if got.BuildDate != tt.want.BuildDate {
				t.Errorf("GetVersion().BuildDate = %q, want %q", got.BuildDate, tt.want.BuildDate)
			}
			if got.Tag != tt.want.Tag {
				t.Errorf("GetVersion().Tag = %q, want %q", got.Tag, tt.want.Tag)
			}
			if got.GoVersion == "" {
				t.Error("GetVersion().GoVersion is empty")
			}
			if got.Compiler == "" {
				t.Error("GetVersion().Compiler is empty")
			}
			if got.Platform == "" {
				t.Error("GetVersion().Platform is empty")
			}
		})
	}
}

func TestCheckNewVersion(t *testing.T) {
	got := CheckNewVersion()
	if got != false {
		t.Errorf("CheckNewVersion() = %v, want false", got)
	}
}
