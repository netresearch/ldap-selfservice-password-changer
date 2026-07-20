/**
 * Forgot password initialization wrapper
 * for CSP compliance (no inline scripts).
 *
 * NOTE: `document.currentScript` is ALWAYS null inside ES modules (per spec),
 * so we look the script up explicitly by src.
 */

import { init } from "./forgot-password.js";

// Selected by its config attribute (always rendered by the template) rather
// than by src, so renaming or cache-busting the bundle cannot break it.
const currentScript = document.querySelector<HTMLScriptElement>("script[data-reset-identifier-mode]");
const mode = currentScript?.dataset["resetIdentifierMode"] ?? "email";

// Initialize forgot password functionality
init(mode);
