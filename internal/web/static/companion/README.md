# Local animated companions

Procedural keyholder and wizard gophers, copied from the
internal companion prototype at commit 6faf9f602c24eef67f5d1502ac72ab84e9f24fe6
(2026-09-22). Adaptations include an external shadow stylesheet for strict CSP,
English/German labels, keyring hit testing and correct canvas viewport sizing.
The component does not read form fields or call any network service.

Load `avatar.css` in the page and `login-companion.js` as a same-origin ES module.
Use `<login-companion character="keyholder">` with an image child for no-JavaScript,
module-failure and WebGL-failure fallback. Without an image child, WebGL failure
uses a static SVG. No pause button is enabled in these integrations; the system
reduced-motion preference renders a still pose. Hidden/offscreen components stop
requesting frames and disconnect disposes GPU resources. Gestures have replay
guards; the keyholder retains its subtle eye quirks and reduced blink frequency.

Three.js 0.186.1 is bundled under its MIT license (`vendor/THREE-LICENSE.txt`).
Unused robot geometry and its Netresearch symbol are omitted. The host page
provides the existing application logo as its fallback; no extra brand asset is bundled. Browser regression checks live in `tests/companion/` at the
repository root; their README explains how to run them without real accounts.

The default password-change page loads `keyholder-login.js`. Clicking the keys or activating the figure by keyboard jingles them. The existing form emits `password-change-start` after validation and bot verification, immediately before its unchanged RPC request. The adapter only starts the gesture; it does not delay the request. Custom operator branding and the reset pages retain their existing images.

## Updating the renderer

The upstream npm package `three@0.186.1` supplies `build/three.module.js`,
`build/three.core.js` and `LICENSE`, copied unchanged into `vendor/`.
Both modules are required and served locally; no CDN or import map is used.
Source: https://registry.npmjs.org/three/0.186.1 (checked 2026-09-29).
The package archive was verified against its registry SHA-512 integrity value.
When upgrading, copy both modules and the license from the same release, then run
Go asset tests and `tests/companion/` browser checks. Three.js now requires WebGL 2;
unsupported browsers retain the host application's original image and form behaviour.
