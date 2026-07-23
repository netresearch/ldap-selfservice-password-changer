package static

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// MaxOverrideBytes caps a single branding file. Logos and icons are small; the
// cap exists so a mistyped path (a device node, a huge archive) fails at
// startup instead of being read into memory on first request.
const MaxOverrideBytes = 2 << 20 // 2 MiB

// DarkLogo is the only overridable asset with no embedded counterpart. It is
// served in dark mode when supplied; without it the regular logo is used for
// both themes.
const DarkLogo = "logo-dark.webp"

// overridableAssets lists the files a branding directory may replace.
//
// Deliberately limited to imagery and icon metadata: styles.css and js/ are
// excluded because overriding them would let a deployment silently break the
// accessibility guarantees (focus rings, contrast, live regions) that the
// templates depend on, and would turn the branding directory into a script
// injection point. Anything not listed here is rejected at startup rather
// than ignored, so a typo surfaces instead of silently doing nothing.
var overridableAssets = map[string]struct{}{
	"logo.webp":                  {},
	DarkLogo:                     {},
	"favicon.ico":                {},
	"favicon-16x16.png":          {},
	"favicon-32x32.png":          {},
	"apple-touch-icon.png":       {},
	"android-chrome-192x192.png": {},
	"android-chrome-512x512.png": {},
	"mstile-150x150.png":         {},
	"safari-pinned-tab.svg":      {},
	"site.webmanifest":           {},
	"browserconfig.xml":          {},
}

// OverridableAssets returns the sorted set of file names a branding directory
// may contain, for error messages and documentation.
func OverridableAssets() []string {
	names := make([]string, 0, len(overridableAssets))
	for name := range overridableAssets {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}

// ValidateOverlay checks that dir is usable as a branding directory and
// reports whether it supplies a dark-mode logo.
//
// Every problem is reported at startup: a missing directory, an unknown file
// name, a non-regular file (a directory, a symlink to one, a device node), or
// a file above MaxOverrideBytes. An empty dir is valid and means "no
// overrides".
func ValidateOverlay(dir string) (bool, error) {
	if dir == "" {
		return false, nil
	}

	var hasDarkLogo bool

	info, err := os.Stat(dir)
	if err != nil {
		return false, fmt.Errorf("branding directory %q: %w", dir, err)
	}
	if !info.IsDir() {
		return false, fmt.Errorf("branding directory %q is not a directory (mode %s)", dir, info.Mode())
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, fmt.Errorf("read branding directory %q: %w", dir, err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if _, ok := overridableAssets[name]; !ok {
			return false, fmt.Errorf(
				"branding directory %q contains unexpected file %q; overridable assets are %v",
				dir, name, OverridableAssets(),
			)
		}

		// Stat rather than entry.Info so a symlink is resolved: a symlink to a
		// directory or a device node must be rejected, not silently served.
		fi, statErr := os.Stat(filepath.Join(dir, name))
		if statErr != nil {
			return false, fmt.Errorf("branding asset %q: %w", name, statErr)
		}
		if !fi.Mode().IsRegular() {
			return false, fmt.Errorf("branding asset %q is not a regular file (mode %s)", name, fi.Mode())
		}
		if fi.Size() > MaxOverrideBytes {
			return false, fmt.Errorf(
				"branding asset %q is %d bytes, above the %d byte limit",
				name, fi.Size(), MaxOverrideBytes,
			)
		}

		if name == DarkLogo {
			hasDarkLogo = true
		}
	}

	return hasDarkLogo, nil
}

// overlayFS serves files from an operator-supplied directory, falling back to
// the embedded assets for anything the directory does not override.
type overlayFS struct {
	dir      string
	embedded fs.FS
}

// NewOverlay returns a filesystem that layers dir over the embedded assets.
// An empty dir yields the embedded assets unchanged. Call ValidateOverlay
// first; this constructor assumes the directory has already been vetted.
func NewOverlay(dir string) fs.FS {
	if dir == "" {
		return Static
	}

	return overlayFS{dir: dir, embedded: Static}
}

// Open resolves name against the overlay directory first, then the embedded
// assets. Only allowlisted names are looked up in the overlay, so a traversal
// attempt or a request for an unrelated file can never escape into the
// operator's filesystem.
func (o overlayFS) Open(name string) (fs.File, error) {
	if _, ok := overridableAssets[name]; ok {
		// #nosec G304 -- name is not user input: it has just been matched
		// against the fixed allowlist above, so no caller-supplied path or
		// traversal sequence can reach this Open.
		f, err := os.Open(filepath.Join(o.dir, name))
		if err == nil {
			return f, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("open branding asset %q: %w", name, err)
		}
	}

	f, err := o.embedded.Open(name)
	if err != nil {
		return nil, fmt.Errorf("open embedded asset %q: %w", name, err)
	}

	return f, nil
}
