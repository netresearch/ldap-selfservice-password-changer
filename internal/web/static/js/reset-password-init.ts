/**
 * Reset password initialization wrapper - reads configuration from data attributes
 * for CSP compliance (no inline scripts)
 */

import { init } from "./reset-password.js";

// Get the current script element to read data attributes
const currentScript = document.currentScript as HTMLScriptElement | null;

if (currentScript) {
  // Read configuration from data attributes
  const config = {
    minLength: Number(currentScript.dataset["minLength"] ?? "8"),
    minNumbers: Number(currentScript.dataset["minNumbers"] ?? "0"),
    minSymbols: Number(currentScript.dataset["minSymbols"] ?? "0"),
    minUppercase: Number(currentScript.dataset["minUppercase"] ?? "0"),
    minLowercase: Number(currentScript.dataset["minLowercase"] ?? "0")
  };

  // Initialize with configuration
  init(config);
}
