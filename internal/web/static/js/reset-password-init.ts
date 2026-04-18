/**
 * Reset password initialization wrapper — reads configuration from data
 * attributes for CSP compliance (no inline scripts).
 *
 * NOTE: `document.currentScript` is ALWAYS null inside ES modules (per spec),
 * so we look the script up explicitly by src.
 */

import { init } from "./reset-password.js";

const currentScript = document.querySelector<HTMLScriptElement>('script[src*="reset-password-init.js"]');

const config = {
  minLength: Number(currentScript?.dataset["minLength"] ?? "8"),
  minNumbers: Number(currentScript?.dataset["minNumbers"] ?? "0"),
  minSymbols: Number(currentScript?.dataset["minSymbols"] ?? "0"),
  minUppercase: Number(currentScript?.dataset["minUppercase"] ?? "0"),
  minLowercase: Number(currentScript?.dataset["minLowercase"] ?? "0")
};

init(config);
