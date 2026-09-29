# Companion browser checks

From the repository root, run `python3 -m http.server 8080 --bind 127.0.0.1`
and open `http://127.0.0.1:8080/tests/companion/` in a WebGL-capable browser.
The page must finish with `PASS` and no failed assertions.

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
