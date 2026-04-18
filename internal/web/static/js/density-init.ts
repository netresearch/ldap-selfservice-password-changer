/**
 * Density initialization - runs before page renders to prevent layout shift.
 * Extracted from inline script for CSP compliance.
 *
 * Sets `data-density` on <html> synchronously. The CSS variants in
 * tailwind.css match `[data-density="..."] *`, so every descendant picks
 * up the right spacing/visibility at first paint.
 */

function initDensity(): void {
  const storedDensityMode = localStorage.getItem("densityMode") ?? "auto";

  const determineDensity = (): string => {
    const isTouch = window.matchMedia("(pointer: coarse)").matches;
    const highContrast = window.matchMedia("(prefers-contrast: more)").matches;
    return isTouch || highContrast ? "comfortable" : "compact";
  };

  const actualDensity = storedDensityMode === "auto" ? determineDensity() : storedDensityMode;

  // Single source of truth: <html data-density="...">
  document.documentElement.dataset["density"] = actualDensity;
  // Kept for any selectors that key off the class (none currently, but cheap).
  document.documentElement.classList.toggle("compact", actualDensity === "compact");
}

initDensity();
