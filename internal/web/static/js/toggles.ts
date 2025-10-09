// Shared theme and density toggle functionality

export type ThemeMode = "light" | "dark" | "auto";
export type DensityMode = "auto" | "comfortable" | "compact";
export type DensityState = "comfortable" | "compact"; // actual applied state

/**
 * Initialize theme toggle functionality
 * Handles three-state theme cycling: auto → light → dark → auto
 */
export const initThemeToggle = () => {
  const themeToggle = document.getElementById("themeToggle");
  const themeLightIcon = document.getElementById("theme-light");
  const themeDarkIcon = document.getElementById("theme-dark");
  const themeAutoIcon = document.getElementById("theme-auto");

  if (themeToggle && themeLightIcon && themeDarkIcon && themeAutoIcon) {
    // Initialize button state from localStorage
    const storedTheme = (localStorage.getItem("theme") || "auto") as ThemeMode;

    const applyTheme = (theme: ThemeMode) => {
      themeToggle.dataset["theme"] = theme;

      // Update icon visibility
      themeLightIcon.classList.toggle("hidden", theme !== "light");
      themeDarkIcon.classList.toggle("hidden", theme !== "dark");
      themeAutoIcon.classList.toggle("hidden", theme !== "auto");

      // Update ARIA label to communicate current state and next action
      const nextTheme: ThemeMode = theme === "auto" ? "light" : theme === "light" ? "dark" : "auto";
      const themeLabels = {
        light: "Light mode",
        dark: "Dark mode",
        auto: "Auto mode (follows system preference)"
      };
      themeToggle.setAttribute(
        "aria-label",
        `Theme: ${themeLabels[theme]}. Click to switch to ${themeLabels[nextTheme]}`
      );

      // Apply theme
      if (theme === "light") {
        document.documentElement.classList.remove("dark");
        localStorage.setItem("theme", "light");
      } else if (theme === "dark") {
        document.documentElement.classList.add("dark");
        localStorage.setItem("theme", "dark");
      } else {
        // Auto: follow OS preference
        const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
        document.documentElement.classList.toggle("dark", prefersDark);
        localStorage.setItem("theme", "auto");
      }
    };

    // Initialize on load
    applyTheme(storedTheme);

    // Cycle through states: auto -> light -> dark -> auto
    themeToggle.addEventListener("click", () => {
      const currentTheme = (themeToggle.dataset["theme"] || "auto") as ThemeMode;
      const nextTheme: ThemeMode = currentTheme === "auto" ? "light" : currentTheme === "light" ? "dark" : "auto";
      applyTheme(nextTheme);
    });
  }
};

/**
 * Initialize density toggle functionality
 * Handles three-state density cycling: auto → comfortable → compact → auto
 */
export const initDensityToggle = () => {
  const densityToggle = document.getElementById("densityToggle");
  const densityComfortableIcon = document.getElementById("density-comfortable");
  const densityCompactIcon = document.getElementById("density-compact");
  const densityAutoIcon = document.getElementById("density-auto");
  const mainContainer = document.querySelector<HTMLDivElement>("div[data-density]");

  if (densityToggle && densityComfortableIcon && densityCompactIcon && densityAutoIcon && mainContainer) {
    // Determine appropriate density based on system preferences
    const determineDensityFromPreferences = (): DensityState => {
      const isTouch = window.matchMedia("(pointer: coarse)").matches;
      const highContrast = window.matchMedia("(prefers-contrast: more)").matches;

      // Touch devices or high contrast mode → comfortable (more space, better accessibility)
      if (isTouch || highContrast) return "comfortable";

      // Desktop with mouse and normal contrast → compact (space-efficient)
      return "compact";
    };

    // Apply density mode and update UI
    const applyDensity = (mode: DensityMode) => {
      // Store user's mode preference (not the applied state)
      densityToggle.dataset["densityMode"] = mode;
      localStorage.setItem("densityMode", mode);

      // Determine actual state to apply
      const actualState: DensityState = mode === "auto" ? determineDensityFromPreferences() : mode;

      // Apply to main container (this triggers all density-variant classes)
      mainContainer.dataset["density"] = actualState;

      // Apply to html element (this triggers all density-variant classes)
      document.documentElement.classList.remove("compact");
      if (actualState === "compact") {
        document.documentElement.classList.add("compact");
      }

      // Update icon visibility (show only current mode's icon)
      densityComfortableIcon.classList.toggle("hidden", mode !== "comfortable");
      densityCompactIcon.classList.toggle("hidden", mode !== "compact");
      densityAutoIcon.classList.toggle("hidden", mode !== "auto");

      // Update ARIA label for accessibility
      const modeLabels = {
        auto: "Auto density (follows system preferences)",
        comfortable: "Comfortable mode (WCAG AAA, spacious layout with all labels)",
        compact: "Compact mode (WCAG AA, simplified layout with hidden labels)"
      };

      const nextMode: DensityMode = mode === "auto" ? "comfortable" : mode === "comfortable" ? "compact" : "auto";

      densityToggle.setAttribute(
        "aria-label",
        `Density: ${modeLabels[mode]}. Click to switch to ${modeLabels[nextMode]}`
      );
    };

    // Initialize on page load
    const storedMode = (localStorage.getItem("densityMode") || "auto") as DensityMode;
    applyDensity(storedMode);

    // Set up reactive monitoring for system preference changes
    const setupPreferenceMonitoring = () => {
      const touchQuery = window.matchMedia("(pointer: coarse)");
      const contrastQuery = window.matchMedia("(prefers-contrast: more)");

      const updateIfAuto = () => {
        const currentMode = densityToggle.dataset["densityMode"] as DensityMode;
        // Only re-evaluate if in auto mode
        if (currentMode === "auto") {
          applyDensity("auto"); // Re-run preference detection
        }
      };

      // Listen for system preference changes (supported in modern browsers)
      touchQuery.addEventListener?.("change", updateIfAuto);
      contrastQuery.addEventListener?.("change", updateIfAuto);
    };

    setupPreferenceMonitoring();

    // Button click handler - cycle through states
    densityToggle.addEventListener("click", () => {
      const currentMode = (densityToggle.dataset["densityMode"] || "auto") as DensityMode;

      // Cycle: auto → comfortable → compact → auto
      const nextMode: DensityMode =
        currentMode === "auto" ? "comfortable" : currentMode === "comfortable" ? "compact" : "auto";

      applyDensity(nextMode);
    });
  }
};
