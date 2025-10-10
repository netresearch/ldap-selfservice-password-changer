/**
 * Density initialization - runs before page renders to prevent layout shift
 * Extracted from inline script for CSP compliance
 */

export function initDensity(): void {
  const storedDensityMode = localStorage.getItem("densityMode") ?? "auto";

  // Determine actual density to apply
  const determineDensity = (): string => {
    const isTouch = window.matchMedia("(pointer: coarse)").matches;
    const highContrast = window.matchMedia("(prefers-contrast: more)").matches;
    return isTouch || highContrast ? "comfortable" : "compact";
  };

  const actualDensity = storedDensityMode === "auto" ? determineDensity() : storedDensityMode;

  // Apply to HTML element for global CSS variants
  if (actualDensity === "compact") {
    document.documentElement.classList.add("compact");
  } else {
    document.documentElement.classList.remove("compact");
  }

  // Apply to page-card container for density-variant classes
  // Wait for DOM to be ready before querying elements
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", () => {
      const pageCard = document.querySelector<HTMLDivElement>("div[data-density]");
      if (pageCard?.dataset) pageCard.dataset["density"] = actualDensity;
    });
  } else {
    const pageCard = document.querySelector<HTMLDivElement>("div[data-density]");
    if (pageCard?.dataset) pageCard.dataset["density"] = actualDensity;
  }
}

// Run immediately (before DOMContentLoaded to prevent layout shift)
initDensity();
