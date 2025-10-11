/**
 * Theme initialization - runs before page renders to prevent flash
 * Extracted from inline script for CSP compliance
 */

function initTheme(): void {
  // Initialize theme before page renders to prevent flash
  const storedTheme = localStorage.getItem("theme");
  const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;

  if (storedTheme === "dark") {
    document.documentElement.classList.add("dark");
  } else if (storedTheme === "light") {
    document.documentElement.classList.remove("dark");
  } else if (!storedTheme || storedTheme === "auto") {
    // Auto mode: follow OS preference
    if (prefersDark) {
      document.documentElement.classList.add("dark");
    }
  }
}

// Run immediately (before DOMContentLoaded to prevent flash)
initTheme();
