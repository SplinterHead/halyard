package manager

import (
	"testing"
)

func TestImageMatches(t *testing.T) {
	tests := []struct {
		serviceImage string
		targetRepo   string
		targetTag    string
		expected     bool
	}{
		{"nginx:latest", "nginx", "latest", true},
		{"docker.io/library/nginx:latest", "nginx", "latest", true},
		{"library/nginx:latest", "nginx", "latest", true},
		{"nginx:latest@sha256:1234567890abcdef", "nginx", "latest", true},
		{"nginx", "nginx", "latest", true},
		{"redis:7", "redis", "7", true},
		{"redis:7", "redis", "latest", false},
		{"ghcr.io/splinterhead/halyard-agent:1.1.0", "ghcr.io/splinterhead/halyard-agent", "1.1.0", true},
		{"myorg/app:v1@sha256:deadbeef", "myorg/app", "v1", true},
		{"otherorg/app:v1", "myorg/app", "v1", false},
	}

	for _, tc := range tests {
		actual := imageMatches(tc.serviceImage, tc.targetRepo, tc.targetTag)
		if actual != tc.expected {
			t.Errorf("imageMatches(%q, %q, %q) = %v; want %v",
				tc.serviceImage, tc.targetRepo, tc.targetTag, actual, tc.expected)
		}
	}
}

func TestNormalizeImageRef(t *testing.T) {
	tests := []struct {
		input       string
		expectedRepo string
		expectedTag  string
	}{
		{"nginx", "nginx", "latest"},
		{"nginx:1.25", "nginx", "1.25"},
		{"docker.io/library/ubuntu:22.04", "ubuntu", "22.04"},
		{"ghcr.io/foo/bar:latest@sha256:abc", "ghcr.io/foo/bar", "latest"},
		{"linuxserver/sonarr:latest@sha256:44b06045a98bc1f237b9ffefc4b28dab64cb29305e4a2a94875ae93b5224e4cc", "linuxserver/sonarr", "latest"},
		{"localhost:5000/app:1.0", "localhost:5000/app", "1.0"},
		{"registry.example.com:5000/org/app:v2.1@sha256:deadbeef", "registry.example.com:5000/org/app", "v2.1"},
		{"localhost:5000/app", "localhost:5000/app", "latest"},
	}

	for _, tc := range tests {
		repo, tag := normalizeImageRef(tc.input)
		if repo != tc.expectedRepo || tag != tc.expectedTag {
			t.Errorf("normalizeImageRef(%q) = (%q, %q); want (%q, %q)",
				tc.input, repo, tag, tc.expectedRepo, tc.expectedTag)
		}
	}
}
