package static_test

import (
	"fmt"
	"io"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/web/static"
)

// writeFile creates a file inside dir with the given contents.
func writeFile(t *testing.T, dir, name, contents string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// newOverlay builds an overlay over dir, failing the test if construction does.
func newOverlay(t *testing.T, dir string) fs.FS {
	t.Helper()

	fsys, err := static.NewOverlay(dir)
	if err != nil {
		t.Fatalf("NewOverlay(%q): %v", dir, err)
	}

	return fsys
}

func readAll(t *testing.T, fsys fs.FS, name string) (string, error) {
	t.Helper()

	f, err := fsys.Open(name)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", name, err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			t.Errorf("close %s: %v", name, cerr)
		}
	}()

	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}

	return string(b), nil
}

func TestValidateOverlay_EmptyDirectoryMeansNoOverrides(t *testing.T) {
	hasDark, err := static.ValidateOverlay("")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if hasDark {
		t.Error("expected no dark logo for an unconfigured overlay")
	}
}

func TestValidateOverlay_DetectsDarkLogo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "logo.webp", "light")

	hasDark, err := static.ValidateOverlay(dir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if hasDark {
		t.Error("expected no dark logo before one is supplied")
	}

	writeFile(t, dir, static.DarkLogo, "dark")

	hasDark, err = static.ValidateOverlay(dir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !hasDark {
		t.Error("expected the dark logo to be detected")
	}
}

// Kubernetes materializes ConfigMap and Secret volumes as `..data` plus a
// `..<timestamp>` directory. Rejecting those as "unexpected files" would make
// the branding directory unusable in the deployment topology the feature is
// documented for, and would push operators towards a writable emptyDir — the
// one arrangement in which a planted symlink becomes someone else's problem.
func TestValidateOverlay_AcceptsKubernetesProjectedVolumeLayout(t *testing.T) {
	dir := t.TempDir()

	// kubelet creates the timestamped directory, a RELATIVE `..data` symlink to
	// it, and one relative symlink per key. Relative matters: an absolute
	// target would leave the root and be refused, correctly.
	const stamp = "..2026_07_23_10_00_00.1234"

	dataDir := filepath.Join(dir, stamp)
	if err := os.Mkdir(dataDir, 0o750); err != nil {
		t.Fatalf("mkdir timestamped dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "logo.webp"), []byte("light"), 0o600); err != nil {
		t.Fatalf("write projected logo: %v", err)
	}
	if err := os.Symlink(stamp, filepath.Join(dir, "..data")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Symlink(filepath.Join("..data", "logo.webp"), filepath.Join(dir, "logo.webp")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if _, err := static.ValidateOverlay(dir); err != nil {
		t.Fatalf("expected the projected-volume layout to be accepted, got %v", err)
	}

	// The key symlink stays inside the directory, so it must still serve.
	got, err := readAll(t, newOverlay(t, dir), "logo.webp")
	if err != nil {
		t.Fatalf("open projected logo: %v", err)
	}
	if got != "light" {
		t.Errorf("expected the projected logo contents, got %q", got)
	}
}

func TestValidateOverlay_Rejects(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr string
	}{
		{
			name:    "missing directory",
			setup:   func(t *testing.T) string { t.Helper(); return filepath.Join(t.TempDir(), "absent") },
			wantErr: "branding directory",
		},
		{
			name: "path is a file",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				path := filepath.Join(dir, "not-a-dir")
				writeFile(t, dir, "not-a-dir", "x")

				return path
			},
			wantErr: "is not a directory",
		},
		{
			name: "unknown file name",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				writeFile(t, dir, "styles.css", "body{}")

				return dir
			},
			wantErr: "unexpected file",
		},
		{
			name: "asset is a directory",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				if err := os.Mkdir(filepath.Join(dir, "logo.webp"), 0o750); err != nil {
					t.Fatalf("mkdir: %v", err)
				}

				return dir
			},
			wantErr: "not a regular file",
		},
		{
			name: "asset above the size cap",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				writeFile(t, dir, "logo.webp", strings.Repeat("x", static.MaxOverrideBytes+1))

				return dir
			},
			wantErr: "above the",
		},
		{
			name: "dangling symlink",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				if err := os.Symlink(filepath.Join(dir, "gone"), filepath.Join(dir, "logo.webp")); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}

				return dir
			},
			wantErr: "logo.webp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := static.ValidateOverlay(tt.setup(t))
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not mention %q", err, tt.wantErr)
			}
		})
	}
}

// A file exactly at the cap is allowed; the check must be > and not >=.
func TestValidateOverlay_AcceptsAssetExactlyAtTheSizeCap(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "logo.webp", strings.Repeat("x", static.MaxOverrideBytes))

	if _, err := static.ValidateOverlay(dir); err != nil {
		t.Fatalf("a file exactly at the cap must be accepted, got %v", err)
	}
}

// The startup check must reject a symlink whose target is a regular file
// outside the directory. os.Stat would report the *target's* mode, so an
// IsRegular check alone accepts it — and the file would then be published
// under a public, unauthenticated /static/ URL.
func TestValidateOverlay_RejectsSymlinkEscapingTheDirectory(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(t.TempDir(), "credentials.env")
	if err := os.WriteFile(secret, []byte("LDAP_READONLY_PASSWORD=hunter2"), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}
	if err := os.Symlink(secret, filepath.Join(dir, "favicon.ico")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if _, err := static.ValidateOverlay(dir); err == nil {
		t.Fatal("expected a symlink pointing outside the branding directory to be rejected")
	}
}

func TestValidateOverlay_RejectsSymlinkToDirectory(t *testing.T) {
	dir := t.TempDir()
	target := t.TempDir()
	if err := os.Symlink(target, filepath.Join(dir, "logo.webp")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if _, err := static.ValidateOverlay(dir); err == nil {
		t.Fatal("expected a symlinked directory to be rejected")
	}
}

func TestValidateOverlay_ReportsUnreadableDirectory(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: permission bits do not restrict access")
	}

	dir := t.TempDir()
	writeFile(t, dir, "logo.webp", "light")
	if err := os.Chmod(dir, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(dir, 0o700); err != nil {
			t.Logf("restore mode: %v", err)
		}
	})

	if _, err := static.ValidateOverlay(dir); err == nil {
		t.Fatal("expected an unreadable branding directory to be reported at startup")
	}
}

func TestNewOverlay_WithoutDirectoryServesEmbeddedAssets(t *testing.T) {
	got, err := readAll(t, newOverlay(t, ""), "logo.webp")
	if err != nil {
		t.Fatalf("open embedded logo: %v", err)
	}
	if got == "" {
		t.Error("expected the embedded logo to have content")
	}
}

func TestNewOverlay_ReportsAMissingDirectory(t *testing.T) {
	if _, err := static.NewOverlay(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Fatal("expected NewOverlay to report a missing directory")
	}
}

func TestNewOverlay_OverrideWinsAndOthersFallBack(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "logo.webp", "custom-logo")

	fsys := newOverlay(t, dir)

	got, err := readAll(t, fsys, "logo.webp")
	if err != nil {
		t.Fatalf("open overridden logo: %v", err)
	}
	if got != "custom-logo" {
		t.Errorf("expected the overlay logo, got %q", got)
	}

	embedded, err := readAll(t, fsys, "favicon.ico")
	if err != nil {
		t.Fatalf("open embedded favicon: %v", err)
	}
	if embedded == "" {
		t.Error("expected the embedded favicon to be served")
	}
}

// Only allowlisted names are looked up in the overlay directory. Without this,
// a file dropped next to the branding assets could be served.
func TestNewOverlay_IgnoresOverlayForNonAllowlistedNames(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "styles.css", "body{content:'pwned'}")

	got, err := readAll(t, newOverlay(t, dir), "styles.css")
	if err != nil {
		t.Fatalf("open styles.css: %v", err)
	}
	if strings.Contains(got, "pwned") {
		t.Error("styles.css was served from the overlay directory; it is not overridable")
	}
}

func TestNewOverlay_TraversalCannotEscape(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(filepath.Dir(dir), "secret.txt")
	if err := os.WriteFile(secret, []byte("top-secret"), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}
	t.Cleanup(func() {
		if rerr := os.Remove(secret); rerr != nil {
			t.Logf("remove %s: %v", secret, rerr)
		}
	})

	fsys := newOverlay(t, dir)

	for _, name := range []string{"../secret.txt", "../../secret.txt", "/etc/hostname"} {
		got, err := readAll(t, fsys, name)
		if err == nil && strings.Contains(got, "top-secret") {
			t.Errorf("%q escaped the overlay directory", name)
		}
	}
}

// Startup validation is operator feedback, not the security boundary: the
// directory can change afterwards. Everything ValidateOverlay checked must be
// re-checked when the file is served.
func TestNewOverlay_RecheckRejectsSymlinkPlantedAfterValidation(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "favicon.ico", "icon")

	if _, err := static.ValidateOverlay(dir); err != nil {
		t.Fatalf("directory should validate clean: %v", err)
		fsys := newOverlay(t, dir)

		secret := filepath.Join(t.TempDir(), "credentials.env")
		if err := os.WriteFile(secret, []byte("LDAP_READONLY_PASSWORD=hunter2"), 0o600); err != nil {
			t.Fatalf("write secret: %v", err)
		}
		if err := os.Remove(filepath.Join(dir, "favicon.ico")); err != nil {
			t.Fatalf("remove: %v", err)
		}
		if err := os.Symlink(secret, filepath.Join(dir, "favicon.ico")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}

		got, err := readAll(t, fsys, "favicon.ico")
		if err == nil && strings.Contains(got, "hunter2") {
			t.Error("a symlink planted after startup served a file outside the branding directory")
		}
	}
}

func TestNewOverlay_RecheckRejectsOversizeFileSwappedInAfterValidation(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "favicon.ico", "icon")

	if _, err := static.ValidateOverlay(dir); err != nil {
		t.Fatalf("directory should validate clean: %v", err)
		fsys := newOverlay(t, dir)

		writeFile(t, dir, "favicon.ico", strings.Repeat("x", static.MaxOverrideBytes+1))

		got, err := readAll(t, fsys, "favicon.ico")
		if err != nil {
			t.Fatalf("open favicon: %v", err)
		}
		if len(got) > static.MaxOverrideBytes {
			t.Errorf("served %d bytes, above the cap; the size check is startup-only", len(got))
		}
	}
}

func TestNewOverlay_RecheckDoesNotBlockOnAFifo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "favicon.ico", "icon")

	if _, err := static.ValidateOverlay(dir); err != nil {
		t.Fatalf("directory should validate clean: %v", err)
		fsys := newOverlay(t, dir)

		fifo := filepath.Join(dir, "favicon.ico")
		if err := os.Remove(fifo); err != nil {
			t.Fatalf("remove: %v", err)
		}
		if err := syscall.Mkfifo(fifo, 0o600); err != nil {
			t.Skipf("FIFOs unavailable: %v", err)
		}

		// Without O_NONBLOCK, open(2) on a FIFO with no writer blocks
		// forever: one unauthenticated request would pin a goroutine and a
		// connection for the life of the process.
		done := make(chan struct{})
		go func() {
			defer close(done)
			if _, err := readAll(t, fsys, "favicon.ico"); err != nil {
				t.Logf("fifo read returned %v (the point is that it returned)", err)
			}
		}()

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("opening a FIFO blocked the request; O_NONBLOCK is not in effect")
		}
	}
}

// The dark logo is the only allowlisted asset with no embedded counterpart.
// The pages are rendered once at startup, so if the operator removes the
// variant afterwards the HTML still points at it — serving the light logo
// keeps that from becoming a broken image in dark mode.
func TestNewOverlay_DarkLogoFallsBackToTheLightLogo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "logo.webp", "custom-light")
	writeFile(t, dir, static.DarkLogo, "custom-dark")

	fsys := newOverlay(t, dir)

	got, err := readAll(t, fsys, static.DarkLogo)
	if err != nil {
		t.Fatalf("open dark logo: %v", err)
	}
	if got != "custom-dark" {
		t.Errorf("expected the supplied dark logo, got %q", got)
	}

	if err := os.Remove(filepath.Join(dir, static.DarkLogo)); err != nil {
		t.Fatalf("remove dark logo: %v", err)
	}

	got, err = readAll(t, fsys, static.DarkLogo)
	if err != nil {
		t.Fatalf("dark logo must stay servable after removal: %v", err)
	}
	if got != "custom-light" {
		t.Errorf("expected a fallback to the light logo, got %q", got)
	}
}

// Go's MIME table has no .webmanifest entry and a scratch container has no
// /etc/mime.types, so without an explicit registration the manifest is
// content-sniffed — and a manifest whose bytes look like markup is then served
// as text/html from the application's own origin.
func TestWebmanifestHasAnExplicitMediaType(t *testing.T) {
	got := mime.TypeByExtension(".webmanifest")
	if !strings.HasPrefix(got, "application/manifest+json") {
		t.Errorf("mime.TypeByExtension(\".webmanifest\") = %q, want application/manifest+json", got)
	}
}

func TestOverridableAssets_IsSortedAndIncludesTheDarkLogo(t *testing.T) {
	names := static.OverridableAssets()
	if len(names) == 0 {
		t.Fatal("expected a non-empty allowlist")
	}

	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Errorf("allowlist is not sorted at %d: %q > %q", i, names[i-1], names[i])
		}
	}

	var found bool
	for _, n := range names {
		if n == static.DarkLogo {
			found = true
		}
	}
	if !found {
		t.Errorf("%q missing from the allowlist", static.DarkLogo)
	}
}
