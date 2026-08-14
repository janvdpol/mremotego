package update

import "testing"

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.4.0", "1.3.0", 1},
		{"1.3.0", "1.4.0", -1},
		{"1.4.0", "1.4.0", 0},
		{"1.4.0", "1.4", 0},
		{"2.0.0", "1.99.99", 1},
		{"1.10.0", "1.9.0", 1}, // numeric, not lexicographic, comparison
		{"1.4.0-alpha", "1.4.0", 0},
	}
	for _, c := range cases {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestAssetNameFor(t *testing.T) {
	cases := []struct {
		goos, goarch, want string
	}{
		{"windows", "amd64", "windows-amd64.zip"},
		{"linux", "amd64", "linux-amd64.tar.gz"},
		{"linux", "arm64", "linux-arm64.tar.gz"},
		{"darwin", "arm64", "darwin-arm64.tar.gz"},
	}
	for _, c := range cases {
		if got := assetNameFor(c.goos, c.goarch); got != c.want {
			t.Errorf("assetNameFor(%q, %q) = %q, want %q", c.goos, c.goarch, got, c.want)
		}
	}
}

func TestCheckLatestSkipsDevBuilds(t *testing.T) {
	for _, v := range []string{"", "dev", " dev "} {
		rel, ok, err := CheckLatest(v)
		if rel != nil || ok || err != nil {
			t.Errorf("CheckLatest(%q) = (%v, %v, %v), want (nil, false, nil)", v, rel, ok, err)
		}
	}
}
