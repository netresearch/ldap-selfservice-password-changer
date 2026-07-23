package static

import (
	"fmt"
	"io/fs"
	"mime"
	"os"
	"sort"
	"strings"
	"syscall"
)

// MaxOverrideBytes caps a single branding file. Logos and icons are small; the
// cap keeps a mistyped path (an archive, a log file) from being streamed to
// every visitor. It is enforced both at startup and on every request, because
// the directory can change while the process runs.
const MaxOverrideBytes = 2 << 20 // 2 MiB

// DarkLogo is the only overridable asset with no embedded counterpart. It is
// served in dark mode when supplied; when it is absent the regular logo is
// served in its place, so a dark-mode page never renders a broken image.
const DarkLogo = "logo-dark.webp"

// lightLogo backs DarkLogo when the branding directory does not supply a dark
// variant.
const lightLogo = "logo.webp"

// overridableAssets lists the files a branding directory may replace.
//
// Deliberately limited to imagery and icon metadata: styles.css and js/ are
// excluded because overriding them would let a deployment silently break the
// accessibility guarantees (focus rings, contrast, live regions) that the
// templates depend on, and would turn the branding directory into a script
// injection point. Anything not listed here is rejected at startup rather
// than ignored, so a typo surfaces instead of silently doing nothing.
var overridableAssets = map[string]struct{}{
	lightLogo:                    {},
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

// Registering the media type is a package-level side effect on purpose: the
// manifest is served whether or not a branding directory is configured, so
// there is no constructor that reliably runs first.
//
//nolint:gochecknoinits // must apply before the first request, no other hook
func init() {
	// Go's MIME table has no .webmanifest entry and a scratch-based container
	// has no /etc/mime.types, so without this the manifest is content-sniffed.
	// A manifest whose bytes look like markup would then be served as
	// text/html from the application's own origin.
	if err := mime.AddExtensionType(".webmanifest", "application/manifest+json"); err != nil {
		panic("register .webmanifest media type: " + err.Error())
	}
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
// It exists to turn operator mistakes into a startup error with a readable
// message; it is NOT the security boundary. Because the directory can change
// after this runs, Open re-checks everything it relies on. Entries whose name
// begins with a dot are skipped: Kubernetes materializes ConfigMap and Secret
// volumes as `..data` plus a `..<timestamp>` directory, and rejecting those
// would make the feature unusable in its main deployment topology.
func ValidateOverlay(dir string) (bool, error) {
	if dir == "" {
		return false, nil
	}

	info, err := os.Stat(dir)
	if err != nil {
		return false, fmt.Errorf("branding directory %q: %w", dir, err)
	}
	if !info.IsDir() {
		return false, fmt.Errorf("branding directory %q is not a directory (mode %s)", dir, info.Mode())
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		return false, fmt.Errorf("open branding directory %q: %w", dir, err)
	}
	// Nothing actionable can be done if closing the handle fails during
	// validation; the process either starts or reports the real error above.
	defer func() { _ = root.Close() }() //nolint:errcheck // see comment

	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, fmt.Errorf("read branding directory %q: %w", dir, err)
	}

	var hasDarkLogo bool

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if _, ok := overridableAssets[name]; !ok {
			return false, fmt.Errorf(
				"branding directory %q contains unexpected file %q; overridable assets are %v",
				dir, name, OverridableAssets(),
			)
		}

		// Opening through the root rejects a symlink that leaves the
		// directory, which a plain os.Stat would happily follow: it reports
		// the *target's* mode, so a link to a regular file outside — a
		// credentials file, say — would pass an IsRegular check and then be
		// published under a public /static/ URL.
		if err := checkOverride(root, name); err != nil {
			return false, err
		}

		if name == DarkLogo {
			hasDarkLogo = true
		}
	}

	return hasDarkLogo, nil
}

// checkOverride verifies a single branding file is servable, reporting why
// it is not.
func checkOverride(root *os.Root, name string) error {
	f, err := openConfined(root, name)
	if err != nil {
		return fmt.Errorf("branding asset %q: %w", name, err)
	}
	defer func() { _ = f.Close() }()

	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("branding asset %q: %w", name, err)
	}
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("branding asset %q is not a regular file (mode %s)", name, fi.Mode())
	}
	if fi.Size() > MaxOverrideBytes {
		return fmt.Errorf(
			"branding asset %q is %d bytes, above the %d byte limit",
			name, fi.Size(), MaxOverrideBytes,
		)
	}

	return nil
}

// openConfined opens name relative to root without following any link out of
// it. O_NONBLOCK keeps a FIFO from parking the caller inside open(2) until a
// writer shows up — for a request handler that would pin a goroutine and a
// connection indefinitely, unauthenticated and trivially repeatable.
func openConfined(root *os.Root, name string) (*os.File, error) {
	f, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", name, err)
	}

	return f, nil
}

// overlayFS serves files from an operator-supplied directory, falling back to
// the embedded assets for anything the directory does not override.
type overlayFS struct {
	root     *os.Root
	embedded fs.FS
}

// NewOverlay returns a filesystem that layers dir over the embedded assets.
// An empty dir yields the embedded assets unchanged.
func NewOverlay(dir string) (fs.FS, error) {
	if dir == "" {
		return Static, nil
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("open branding directory %q: %w", dir, err)
	}

	return overlayFS{root: root, embedded: Static}, nil
}

// Open resolves name against the overlay directory first, then the embedded
// assets. Only allowlisted names are looked up in the overlay, and the lookup
// is confined to that directory, so neither a traversal attempt nor a symlink
// planted inside it can reach the rest of the filesystem.
//
// Every guarantee ValidateOverlay checked at startup is re-checked here,
// because the directory may have changed since. An override that is no longer
// servable falls back to the built-in asset rather than failing the request:
// a half-broken rebrand should not take the login page down.
func (o overlayFS) Open(name string) (fs.File, error) {
	if _, ok := overridableAssets[name]; ok {
		if f, ok := o.openOverride(name); ok {
			return f, nil
		}
	}

	// The dark logo is the one allowlisted asset with no built-in version.
	// Serving the light logo keeps a dark-mode page whose operator removed the
	// variant after startup from rendering a broken image, since the rendered
	// HTML still references it.
	if name == DarkLogo {
		return o.Open(lightLogo)
	}

	f, err := o.embedded.Open(name)
	if err != nil {
		return nil, fmt.Errorf("open embedded asset %q: %w", name, err)
	}

	return f, nil
}

// openOverride returns the operator-supplied file, or false when the overlay
// cannot serve it — absent, escaping the directory, not a regular file, or
// grown past the size cap since startup.
func (o overlayFS) openOverride(name string) (fs.File, bool) {
	f, err := openConfined(o.root, name)
	if err != nil {
		return nil, false
	}

	fi, err := f.Stat()
	if err != nil || !fi.Mode().IsRegular() || fi.Size() > MaxOverrideBytes {
		_ = f.Close()

		return nil, false
	}

	return f, true
}
