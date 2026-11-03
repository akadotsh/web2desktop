package builder

import "testing"

func TestParsePlatforms(t *testing.T) {
	platforms, err := ParsePlatforms("darwin/universal,windows/amd64,linux/arm64,windows/amd64")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"darwin/universal", "windows/amd64", "linux/arm64"}
	if len(platforms) != len(want) {
		t.Fatalf("got %d platforms, want %d", len(platforms), len(want))
	}
	for i, platform := range platforms {
		if got := platform.String(); got != want[i] {
			t.Errorf("platform %d = %q, want %q", i, got, want[i])
		}
	}
}

func TestParsePlatformsRejectsUnsupportedTarget(t *testing.T) {
	if _, err := ParsePlatforms("windows/386"); err == nil {
		t.Fatal("expected an error")
	}
}
