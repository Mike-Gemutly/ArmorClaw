package sidecar

import (
	"strings"
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    *Version
		wantErr bool
	}{
		{
			name:    "valid version",
			version: "1.0.0",
			want:    &Version{Major: 1, Minor: 0, Patch: 0},
			wantErr: false,
		},
		{
			name:    "valid version with larger numbers",
			version: "2.5.10",
			want:    &Version{Major: 2, Minor: 5, Patch: 10},
			wantErr: false,
		},
		{
			name:    "invalid format - missing patch",
			version: "1.0",
			wantErr: true,
		},
		{
			name:    "invalid format - too many parts",
			version: "1.0.0.0",
			wantErr: true,
		},
		{
			name:    "invalid format - non-numeric",
			version: "a.b.c",
			wantErr: true,
		},
		{
			name:    "invalid format - empty string",
			version: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.String() != tt.want.String() {
				t.Errorf("ParseVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVersionCompare(t *testing.T) {
	tests := []struct {
		name  string
		v     *Version
		other *Version
		want  int
	}{
		{
			name:  "equal versions",
			v:     &Version{Major: 1, Minor: 0, Patch: 0},
			other: &Version{Major: 1, Minor: 0, Patch: 0},
			want:  0,
		},
		{
			name:  "v less - major version",
			v:     &Version{Major: 1, Minor: 0, Patch: 0},
			other: &Version{Major: 2, Minor: 0, Patch: 0},
			want:  -1,
		},
		{
			name:  "v greater - major version",
			v:     &Version{Major: 2, Minor: 0, Patch: 0},
			other: &Version{Major: 1, Minor: 0, Patch: 0},
			want:  1,
		},
		{
			name:  "v less - minor version",
			v:     &Version{Major: 1, Minor: 0, Patch: 0},
			other: &Version{Major: 1, Minor: 1, Patch: 0},
			want:  -1,
		},
		{
			name:  "v greater - minor version",
			v:     &Version{Major: 1, Minor: 1, Patch: 0},
			other: &Version{Major: 1, Minor: 0, Patch: 0},
			want:  1,
		},
		{
			name:  "v less - patch version",
			v:     &Version{Major: 1, Minor: 0, Patch: 0},
			other: &Version{Major: 1, Minor: 0, Patch: 1},
			want:  -1,
		},
		{
			name:  "v greater - patch version",
			v:     &Version{Major: 1, Minor: 0, Patch: 1},
			other: &Version{Major: 1, Minor: 0, Patch: 0},
			want:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.Compare(tt.other); got != tt.want {
				t.Errorf("Version.Compare() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVersionIsCompatible(t *testing.T) {
	tests := []struct {
		name    string
		v       *Version
		min     string
		max     string
		want    bool
		wantErr bool
	}{
		{
			name:    "compatible - exact min",
			v:       &Version{Major: 1, Minor: 0, Patch: 0},
			min:     "1.0.0",
			max:     "1.5.0",
			want:    true,
			wantErr: false,
		},
		{
			name:    "compatible - exact max",
			v:       &Version{Major: 1, Minor: 5, Patch: 0},
			min:     "1.0.0",
			max:     "1.5.0",
			want:    true,
			wantErr: false,
		},
		{
			name:    "compatible - in range",
			v:       &Version{Major: 1, Minor: 2, Patch: 3},
			min:     "1.0.0",
			max:     "1.5.0",
			want:    true,
			wantErr: false,
		},
		{
			name:    "incompatible - below min",
			v:       &Version{Major: 0, Minor: 9, Patch: 99},
			min:     "1.0.0",
			max:     "1.5.0",
			want:    false,
			wantErr: false,
		},
		{
			name:    "incompatible - above max",
			v:       &Version{Major: 2, Minor: 0, Patch: 0},
			min:     "1.0.0",
			max:     "1.5.0",
			want:    false,
			wantErr: false,
		},
		{
			name:    "invalid min version",
			v:       &Version{Major: 1, Minor: 0, Patch: 0},
			min:     "invalid",
			max:     "1.5.0",
			wantErr: true,
		},
		{
			name:    "invalid max version",
			v:       &Version{Major: 1, Minor: 0, Patch: 0},
			min:     "1.0.0",
			max:     "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.v.IsCompatible(tt.min, tt.max)
			if (err != nil) != tt.wantErr {
				t.Errorf("Version.IsCompatible() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Version.IsCompatible() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckServerCompatibility(t *testing.T) {
	tests := []struct {
		name        string
		serverVer   string
		wantErr     bool
		errContains string
	}{
		{
			name:      "compatible version",
			serverVer: "1.0.0",
			wantErr:   false,
		},
		{
			name:      "compatible version in range",
			serverVer: "1.2.3",
			wantErr:   false,
		},
		{
			name:      "compatible version at max",
			serverVer: "1.5.0",
			wantErr:   false,
		},
		{
			name:        "incompatible - too old",
			serverVer:   "0.9.0",
			wantErr:     true,
			errContains: "version incompatibility",
		},
		{
			name:        "incompatible - too new",
			serverVer:   "2.0.0",
			wantErr:     true,
			errContains: "version incompatibility",
		},
		{
			name:        "invalid version format",
			serverVer:   "invalid",
			wantErr:     true,
			errContains: "invalid server version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckServerCompatibility(tt.serverVer)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckServerCompatibility() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("CheckServerCompatibility() error = %v, wantContains %v", err.Error(), tt.errContains)
				}
			}
		})
	}
}

func TestGetCurrentVersion(t *testing.T) {
	version := GetCurrentVersion()
	if version != ClientVersion {
		t.Errorf("GetCurrentVersion() = %v, want %v", version, ClientVersion)
	}
}

func TestGetSupportedVersionRange(t *testing.T) {
	min, max := GetSupportedVersionRange()
	if min != MinServerVersion {
		t.Errorf("GetSupportedVersionRange() min = %v, want %v", min, MinServerVersion)
	}
	if max != MaxServerVersion {
		t.Errorf("GetSupportedVersionRange() max = %v, want %v", max, MaxServerVersion)
	}
}
