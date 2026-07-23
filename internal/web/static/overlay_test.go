package static_test

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/web/static"
)

// writeFile creates a file inside dir with the given contents.
func writeFile(t *testing.T, dir, name, contents string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
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

// A symlink pointing at a directory must be rejected: os.ReadDir reports the
// link itself, so only an explicit Stat catches it.
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

func TestNewOverlay_WithoutDirectoryServesEmbeddedAssets(t *testing.T) {
	fsys := static.NewOverlay("")

	got, err := readAll(t, fsys, "logo.webp")
	if err != nil {
		t.Fatalf("open embedded logo: %v", err)
	}
	if got == "" {
		t.Error("expected the embedded logo to have content")
	}
}

func TestNewOverlay_OverrideWinsAndOthersFallBack(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "logo.webp", "custom-logo")

	fsys := static.NewOverlay(dir)

	got, err := readAll(t, fsys, "logo.webp")
	if err != nil {
		t.Fatalf("open overridden logo: %v", err)
	}
	if got != "custom-logo" {
		t.Errorf("expected the overlay logo, got %q", got)
	}

	// Not overridden: must still come from the embedded assets.
	embedded, err := readAll(t, fsys, "favicon.ico")
	if err != nil {
		t.Fatalf("open embedded favicon: %v", err)
	}
	if embedded == "" {
		t.Error("expected the embedded favicon to be served")
	}
}

// Only allowlisted names are looked up in the overlay directory. Without this,
// a file dropped next to the branding assets could be served, and a traversal
// attempt could escape the directory entirely.
func TestNewOverlay_IgnoresOverlayForNonAllowlistedNames(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "styles.css", "body{content:'pwned'}")

	fsys := static.NewOverlay(dir)

	got, err := readAll(t, fsys, "styles.css")
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

	fsys := static.NewOverlay(dir)

	for _, name := range []string{"../secret.txt", "../../secret.txt", "/etc/hostname"} {
		got, err := readAll(t, fsys, name)
		if err == nil && strings.Contains(got, "top-secret") {
			t.Errorf("%q escaped the overlay directory", name)
		}
		if err != nil && !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, fs.ErrInvalid) {
			t.Logf("%q rejected with %v", name, err)
		}
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
