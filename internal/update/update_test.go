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

func TestFindChecksum(t *testing.T) {
	listing := "aaaa111  linux-amd64.tar.gz\n" +
		"bbbb222 *windows-amd64.zip\n" + // sha256sum binary-mode marker
		"cccc333  darwin-arm64.tar.gz\n"

	cases := []struct {
		name    string
		want    string
		wantErr bool
	}{
		{"linux-amd64.tar.gz", "aaaa111", false},
		{"windows-amd64.zip", "bbbb222", false}, // leading "*" stripped
		{"darwin-arm64.tar.gz", "cccc333", false},
		{"linux-arm64.tar.gz", "", true}, // not present
	}
	for _, c := range cases {
		got, err := findChecksum(listing, c.name)
		if c.wantErr {
			if err == nil {
				t.Errorf("findChecksum(%q) = %q, want error", c.name, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("findChecksum(%q) unexpected error: %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("findChecksum(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestVerifyChecksumFailsClosedWithoutURL(t *testing.T) {
	err := verifyChecksum("irrelevant-path", &Release{TagName: "v1.6.0", AssetName: "linux-amd64.tar.gz"})
	if err == nil {
		t.Fatal("verifyChecksum with no ChecksumsURL should fail closed, got nil error")
	}
}
