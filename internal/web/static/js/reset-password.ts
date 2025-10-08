import {
  mustBeLongerThan,
  mustIncludeLowercase,
  mustIncludeNumbers,
  mustIncludeSymbols,
  mustIncludeUppercase,
  mustMatchNewPassword,
  mustNotBeEmpty
} from "./validators.js";

type Opts = {
  minLength: number;
  minNumbers: number;
  minSymbols: number;
  minUppercase: number;
  minLowercase: number;
};

export const init = (opts: Opts) => {
  // Theme toggle functionality - three states: light, dark, auto
  const themeToggle = document.getElementById("themeToggle");
  const themeLightIcon = document.getElementById("theme-light");
  const themeDarkIcon = document.getElementById("theme-dark");
  const themeAutoIcon = document.getElementById("theme-auto");

  if (themeToggle && themeLightIcon && themeDarkIcon && themeAutoIcon) {
    // Initialize button state from localStorage
    const storedTheme = localStorage.getItem("theme") || "auto";
    const applyTheme = (theme: "light" | "dark" | "auto") => {
      // Update icon visibility by updating the html element class
      document.documentElement.classList.remove("theme-light", "theme-dark", "theme-auto");
      document.documentElement.classList.add(`theme-${theme}`);

      // Update ARIA label to communicate current state and next action
      const nextTheme = theme === "auto" ? "light" : theme === "light" ? "dark" : "auto";
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
    applyTheme(storedTheme as "light" | "dark" | "auto");

    // Cycle through states: auto -> light -> dark -> auto
    themeToggle.addEventListener("click", () => {
      const currentTheme = document.documentElement.classList.contains("theme-light")
        ? "light"
        : document.documentElement.classList.contains("theme-dark")
          ? "dark"
          : "auto";
      const nextTheme = currentTheme === "auto" ? "light" : currentTheme === "light" ? "dark" : "auto";
      applyTheme(nextTheme);
    });
  }

  // Three-state density toggle: auto, comfortable, compact
  type DensityMode = "auto" | "comfortable" | "compact";
  type DensityState = "comfortable" | "compact"; // actual applied state

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
        const currentMode = (localStorage.getItem("densityMode") || "auto") as DensityMode;
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
      const currentMode = (localStorage.getItem("densityMode") || "auto") as DensityMode;

      // Cycle: auto → comfortable → compact → auto
      const nextMode: DensityMode =
        currentMode === "auto" ? "comfortable" : currentMode === "comfortable" ? "compact" : "auto";

      applyDensity(nextMode);
    });
  }

  const successContainer = document.querySelector<HTMLDivElement>("div[data-purpose='successContainer']");
  if (!successContainer) throw new Error("Could not find success container element");

  const form = document.querySelector<HTMLFormElement>("#form");
  if (!form) throw new Error("Could not find form element");

  const submitButton = form.querySelector<HTMLButtonElement>("& > div[data-purpose='submit'] > button[type='submit']");
  if (!submitButton) throw new Error("Could not find submit button element");

  const submitErrorContainer = form.querySelector<HTMLDivElement>(
    "& > div[data-purpose='submit'] > div[data-purpose='errors']"
  );
  if (!submitErrorContainer) throw new Error("Could not find submit error container element");

  // Extract token from URL
  const urlParams = new URLSearchParams(window.location.search);
  const token = urlParams.get("token");

  if (!token) {
    submitErrorContainer.innerText = "Invalid or missing reset token. Please request a new password reset link.";
    submitButton.disabled = true;
    return;
  }

  type Field = [string, ((v: string) => string)[]];

  const fieldsWithValidators = [
    [
      "new_password",
      [
        mustNotBeEmpty("New Password"),
        mustBeLongerThan(opts.minLength, "New Password"),
        mustIncludeNumbers(opts.minNumbers, "New Password"),
        mustIncludeSymbols(opts.minSymbols, "New Password"),
        mustIncludeUppercase(opts.minUppercase, "New Password"),
        mustIncludeLowercase(opts.minLowercase, "New Password")
      ]
    ],
    ["confirm_password", [mustNotBeEmpty("Confirm New Password"), mustMatchNewPassword("Confirm New Password")]]
  ] satisfies Field[];

  const fields = fieldsWithValidators.map(([name, validators]) => {
    const f = form.querySelector<HTMLDivElement>(`#${name}`);
    if (!f) throw new Error(`Field "${name}" does not exist`);

    const inputContainer = f.querySelector<HTMLDivElement>('div[data-purpose="inputContainer"]');
    if (!inputContainer) throw new Error(`Input container for "${name}" does not exist`);

    const input = inputContainer.querySelector<HTMLInputElement>("input");
    if (!input) throw new Error(`Input for "${name}" does not exist`);

    const revealButton = inputContainer.querySelector<HTMLButtonElement>('button[data-purpose="reveal"]');
    if (!revealButton && input.type === "password") throw new Error(`Reveal button for "${name}" does not exist`);

    const errorContainer = f.querySelector<HTMLDivElement>('div[data-purpose="errors"]');
    if (!errorContainer) throw new Error(`Error for "${name}" does not exist`);

    const getValue = () => input.value;
    const setErrors = (errors: string[]) => {
      errorContainer.innerHTML = "";

      if (errors.length > 0) {
        inputContainer.classList.add("border-red-500");
      } else {
        inputContainer.classList.remove("border-red-500");
      }

      for (const error of errors) {
        const el = document.createElement("p");
        el.innerText = error;

        errorContainer.appendChild(el);
      }
    };

    const validate = () => {
      const value = getValue();

      const errors = validators
        .map((validate) => validate(value))
        .reduce((acc, v) => {
          if (v.length > 0) acc.push(v);

          return acc;
        }, [] as string[]);

      console.log(`Validated "${name}": ${errors.length} error(s)`);

      setErrors(errors);

      return errors.length > 0;
    };

    if (revealButton) {
      revealButton.onclick = (e) => {
        e.preventDefault();
        e.stopPropagation();

        const newType = input.type === "password" ? "text" : "password";
        const revealed = newType === "text";

        console.log(`${revealed ? "Showing" : "Hiding"} content of "${name}"`);

        input.type = newType;
        f.dataset["revealed"] = revealed.toString();
      };
    }

    // Help toggle functionality
    const helpButton = f.querySelector<HTMLButtonElement>('button[data-purpose="help"]');
    const helpText = f.querySelector<HTMLDivElement>('div[data-purpose="helpText"]');

    if (helpButton && helpText) {
      helpButton.onclick = (e) => {
        e.preventDefault();
        e.stopPropagation();

        const isExpanded = helpButton.getAttribute("aria-expanded") === "true";
        const newExpanded = !isExpanded;

        helpButton.setAttribute("aria-expanded", newExpanded.toString());
        helpText.classList.toggle("hidden", !newExpanded);
      };
    }

    return { input, errorContainer, getValue, validate };
  });

  const toggleFields = (enabled: boolean) => {
    [submitButton, ...fields.map(({ input }) => input)].forEach((el) => (el.disabled = !enabled));
    submitButton.dataset["loading"] = (!enabled).toString();
  };

  form.onsubmit = async (e) => {
    e.preventDefault();
    e.stopPropagation();

    const [newPassword] = fields.map((f) => f.getValue());

    const hasErrors = fields.map(({ validate }) => validate()).some((e) => e === true);
    submitButton.disabled = hasErrors;
    if (hasErrors) return;

    console.log("Resetting password...");
    toggleFields(false);

    try {
      const res = await fetch("/api/rpc", {
        method: "POST",
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify({
          method: "reset-password",
          params: [token, newPassword]
        })
      });

      const body = await res.text();

      if (!res.ok) {
        let err = body;

        try {
          const parsed = JSON.parse(body);

          err = parsed.data[0];
        } catch (e) {}

        throw new Error(`An error occurred: ${err}`);
      }

      console.log("Password reset successfully");

      form.style.display = "none";
      successContainer.style.display = "block";
    } catch (e) {
      console.error(e);

      submitErrorContainer.innerText = (e as Error).message;

      // Re-enable inputs but keep the submit button disabled
      toggleFields(true);
      submitButton.disabled = true;
    }
  };

  form.onchange = (e) => {
    e.stopPropagation();

    const hasErrors = fields.map(({ validate }) => validate()).some((e) => e === true);
    submitButton.disabled = hasErrors;
  };
};
