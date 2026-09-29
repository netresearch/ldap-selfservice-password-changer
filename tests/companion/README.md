# Companion browser checks

From the repository root, run `cd tests/companion && npm ci --ignore-scripts && ./node_modules/.bin/playwright install chromium && npm test`.
The runner serves the repository, opens the fixture in headless Chromium with software WebGL and exits non-zero unless the page ends with `PASS`.
The `Companion checks` workflow runs the same command on every pull request. To debug by hand,
serve the repository root with `python3 -m http.server 8080 --bind 127.0.0.1`
and open `http://127.0.0.1:8080/tests/companion/` in a WebGL-capable browser.

These checks exercise the real bundled component and login adapter under a strict
Content Security Policy. They cover the animation random source's boundary values,
canvas proportions, absence of pause buttons,
keyring hit testing or native login replay (where applicable), replay guards,
WebGL failure, original-image fallback, disposal and reduced motion.
The fixture uses no real account, never sends a form request and reads no credentials.

This repository tests only the keyholder adapter; the other application's adapter
is intentionally not bundled. Waits use animation progress or observable form
results with explicit failure deadlines. Gesture settlement allows up to two
minutes for software-rendered browsers; an unmet condition fails the fixture.
