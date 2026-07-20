/**
 * Forgot password initialization wrapper
 * for CSP compliance (no inline scripts).
 *
 * NOTE: `document.currentScript` is ALWAYS null inside ES modules (per spec),
 * so we look the script up explicitly by src.
 */

import { init } from "./forgot-password.js";

const currentScript = document.querySelector<HTMLScriptElement>('script[src*="forgot-password-init.js"]');
const mode = currentScript?.dataset["resetIdentifierMode"] ?? "email";

// Initialize forgot password functionality
init(mode);
