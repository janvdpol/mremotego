// Package update implements self-update: checking GitHub Releases for a newer
// version of MremoteGO, downloading the matching platform archive, and
// replacing the currently running executable with it.
package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// repoOwner/repoName identify the GitHub repository releases are checked against.
const (
	repoOwner = "janvdpol"
	repoName  = "mremotego"
)

var releasesURL = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=10", repoOwner, repoName)

// Release describes a GitHub release that is newer than the running version
// and has a downloadable archive for the current platform.
type Release struct {
	Version   string // e.g. "1.4.0" (no leading "v")
	TagName   string // e.g. "v1.4.0"
	HTMLURL   string
	Notes     string
	AssetURL  string
	AssetName string
}

type ghRelease struct {
	TagName string    `json:"tag_name"`
	HTMLURL string    `json:"html_url"`
	Body    string    `json:"body"`
	Draft   bool      `json:"draft"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// CheckLatest queries GitHub for the most recently published release and
// returns it if newer than currentVersion. It reports ok=false (with no
// error) when already up to date or when currentVersion is a local/dev build.
//
// Every release this project publishes is marked as a GitHub "prerelease"
// (see .github/workflows/build.yml), so the /releases/latest endpoint —
// which explicitly excludes prereleases — always 404s here. The release list
// is queried instead, and the newest non-draft entry is treated as latest
// (GitHub returns the list newest-first).
func CheckLatest(currentVersion string) (rel *Release, ok bool, err error) {
	currentVersion = strings.TrimPrefix(strings.TrimSpace(currentVersion), "v")
	if currentVersion == "" || currentVersion == "dev" {
		return nil, false, nil
	}

	req, err := http.NewRequest(http.MethodGet, releasesURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("github API returned %s", resp.Status)
	}

	var releases []ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, false, err
	}

	for _, r := range releases {
		if r.Draft {
			continue
		}

		latestVersion := strings.TrimPrefix(r.TagName, "v")
		if compareVersions(latestVersion, currentVersion) <= 0 {
			return nil, false, nil // newest published release is not newer than what we have
		}

		assetName := assetNameFor(runtime.GOOS, runtime.GOARCH)
		var assetURL string
		for _, a := range r.Assets {
			if a.Name == assetName {
				assetURL = a.BrowserDownloadURL
				break
			}
		}
		if assetURL == "" {
			return nil, false, fmt.Errorf("release %s has no asset named %q for this platform", r.TagName, assetName)
		}

		return &Release{
			Version:   latestVersion,
			TagName:   r.TagName,
			HTMLURL:   r.HTMLURL,
			Notes:     r.Body,
			AssetURL:  assetURL,
			AssetName: assetName,
		}, true, nil
	}

	return nil, false, nil
}

// assetNameFor returns the release asset name for goos/goarch, matching the
// naming produced by .github/workflows/build.yml (upload-artifact name
// "<goos>-<goarch>", archived as .zip on Windows and .tar.gz elsewhere).
func assetNameFor(goos, goarch string) string {
	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("%s-%s.%s", goos, goarch, ext)
}

// compareVersions compares dotted numeric version strings (e.g. "1.4.0").
// Returns 1 if a > b, -1 if a < b, 0 if equal (non-numeric suffixes on the
// final segment, e.g. "-alpha", are ignored).
func compareVersions(a, b string) int {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	for i := 0; i < len(as) || i < len(bs); i++ {
		var an, bn int
		if i < len(as) {
			an = leadingInt(as[i])
		}
		if i < len(bs) {
			bn = leadingInt(bs[i])
		}
		if an != bn {
			if an > bn {
				return 1
			}
			return -1
		}
	}
	return 0
}

// leadingInt parses the leading run of ASCII digits in s, returning 0 if
// there is none (e.g. for a "0-alpha" style segment).
func leadingInt(s string) int {
	end := 0
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	n, _ := strconv.Atoi(s[:end])
	return n
}

// Apply downloads rel's archive, extracts the executable, replaces the
// currently running binary with it, and relaunches it. On success the
// caller should exit the current process immediately — the app has already
// continued running as the freshly spawned process.
func Apply(rel *Release) error {
	archivePath, err := download(rel.AssetURL, rel.AssetName)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(archivePath)

	newExePath, err := extractExecutable(archivePath)
	if err != nil {
		return fmt.Errorf("extract failed: %w", err)
	}
	defer os.RemoveAll(filepath.Dir(newExePath))

	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine current executable path: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(currentExe); err == nil {
		currentExe = resolved
	}

	if err := replaceExecutable(currentExe, newExePath); err != nil {
		return fmt.Errorf("failed to install update: %w", err)
	}

	if err := relaunch(currentExe); err != nil {
		return fmt.Errorf("update installed, but failed to relaunch automatically — please start MremoteGO again: %w", err)
	}

	return nil
}

// download fetches url to a temporary file (named after assetName, so its
// extension is preserved for extractExecutable) and returns its path.
func download(url, assetName string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: %s", resp.Status)
	}

	tmp, err := os.CreateTemp("", "mremotego-update-*"+filepath.Ext(assetName))
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	return tmp.Name(), nil
}

// extractExecutable pulls the single file out of archivePath (a .zip or
// .tar.gz produced by the release workflow, each containing exactly one
// executable at its root) into a fresh temp directory and returns its path.
func extractExecutable(archivePath string) (string, error) {
	if strings.HasSuffix(archivePath, ".zip") {
		return extractFromZip(archivePath)
	}
	return extractFromTarGz(archivePath)
}

func extractFromZip(archivePath string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		path, err := writeTempExecutable(rc, f.Name)
		rc.Close()
		return path, err
	}
	return "", fmt.Errorf("no file found in update archive")
}

func extractFromTarGz(archivePath string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		return writeTempExecutable(tr, hdr.Name)
	}
	return "", fmt.Errorf("no file found in update archive")
}

func writeTempExecutable(r io.Reader, name string) (string, error) {
	dir, err := os.MkdirTemp("", "mremotego-update")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, filepath.Base(name))
	out, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, r); err != nil {
		return "", err
	}
	return path, nil
}

// relaunch starts exePath as a new detached process with the same arguments
// the current process was given.
func relaunch(exePath string) error {
	cmd := exec.Command(exePath, os.Args[1:]...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

// renameOrCopy moves src to dst. Plain os.Rename can't cross filesystem/volume
// boundaries (common here, since src lives in the system temp directory), so
// on failure this instead copies src into a temp file in dst's own directory
// (guaranteed same filesystem) and renames that into place. Going through a
// same-filesystem rename — rather than writing dst directly — matters when
// dst may be the very executable currently running: overwriting its content
// in place is unsafe, while rename/unlink-and-relink is not.
func renameOrCopy(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	tmp, err := os.CreateTemp(filepath.Dir(dst), ".mremotego-update-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op once successfully renamed away

	in, err := os.Open(src)
	if err != nil {
		tmp.Close()
		return err
	}
	_, copyErr := io.Copy(tmp, in)
	in.Close()
	closeErr := tmp.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, dst); err != nil {
		return err
	}

	_ = os.Remove(src)
	return nil
}

// CleanupOnStartup removes a "<exe>.old" leftover from a previous self-update
// (see the Windows replaceExecutable), retrying for up to ~30s since the old
// process — which spawned this one and then exited — may take a little while
// to fully tear down (Fyne's GL context teardown in particular) and release
// its handle on that file. Callers should invoke this in a goroutine: it's
// meant to finish cleanup shortly after startup, not block it.
func CleanupOnStartup() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	oldPath := exe + ".old"
	if _, err := os.Stat(oldPath); err != nil {
		return // nothing to clean up
	}

	for i := 0; i < 60; i++ {
		if err := os.Remove(oldPath); err == nil {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}
