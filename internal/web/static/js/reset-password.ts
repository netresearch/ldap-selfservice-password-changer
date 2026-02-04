import {
  mustBeLongerThan,
  mustIncludeLowercase,
  mustIncludeNumbers,
  mustIncludeSymbols,
  mustIncludeUppercase,
  mustMatchNewPassword,
  mustNotBeEmpty
} from "./validators.js";
import { initThemeToggle, initDensityToggle } from "./toggles.js";

// SVG error icon for inline use (WCAG 1.4.1 - not relying on color alone)
const ERROR_ICON_SVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="inline-block h-3.5 w-3.5 shrink-0 mr-1" aria-hidden="true"><path fill-rule="evenodd" d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 5a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 5zm0 9a1 1 0 100-2 1 1 0 000 2z" clip-rule="evenodd" /></svg>`;

interface Opts {
  minLength: number;
  minNumbers: number;
  minSymbols: number;
  minUppercase: number;
  minLowercase: number;
}

export const init = (opts: Opts) => {
  // Initialize theme and density toggles
  initThemeToggle();
  initDensityToggle();

  const successContainer = document.querySelector<HTMLDivElement>("div[data-purpose='successContainer']");
  if (!successContainer) throw new Error("Could not find success container element");

  const form = document.querySelector<HTMLFormElement>("#form");
  if (!form) throw new Error("Could not find form element");

  const errorSummary = form.querySelector<HTMLDivElement>("#error-summary");
  if (!errorSummary) throw new Error("Could not find error summary element");

  const errorSummaryText = errorSummary.querySelector<HTMLSpanElement>("span[data-purpose='summaryText']");
  if (!errorSummaryText) throw new Error("Could not find error summary text element");

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
    submitErrorContainer.innerText =
      "Unable to process your request. Please try again or request a new password reset link.";
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

    const revealButton = f.querySelector<HTMLButtonElement>('button[data-purpose="reveal"]');
    if (!revealButton && input.type === "password") throw new Error(`Reveal button for "${name}" does not exist`);

    const errorContainer = f.querySelector<HTMLDivElement>('div[data-purpose="errors"]');
    if (!errorContainer) throw new Error(`Error for "${name}" does not exist`);

    const getValue = () => input.value;
    const setErrors = (errors: string[]) => {
      // Clear existing errors using safe DOM method
      while (errorContainer.firstChild) {
        errorContainer.removeChild(errorContainer.firstChild);
      }

      if (errors.length > 0) {
        inputContainer.classList.add("!border-error-dark", "dark:!border-error-light");
        input.setAttribute("aria-invalid", "true");
      } else {
        inputContainer.classList.remove("!border-error-dark", "dark:!border-error-light");
        input.setAttribute("aria-invalid", "false");
      }

      for (const error of errors) {
        const el = document.createElement("p");
        el.className = "flex items-start gap-1";

        // Create error icon using template element for safe SVG parsing
        // Note: ERROR_ICON_SVG is a hardcoded constant, not user input
        const template = document.createElement("template");
        template.innerHTML = ERROR_ICON_SVG;
        const icon = template.content.firstChild;
        if (icon) el.appendChild(icon);

        // Create text span with textContent which safely escapes all HTML
        const textSpan = document.createElement("span");
        textSpan.textContent = error;
        el.appendChild(textSpan);

        errorContainer.appendChild(el);
      }
    };

    const validate = () => {
      const value = getValue();

      const errors = validators
        .map((validate) => validate(value))
        .reduce<string[]>((acc, v) => {
          if (v.length > 0) acc.push(v);

          return acc;
        }, []);

      console.warn(`Validated "${name}": ${errors.length.toString()} error(s)`);

      setErrors(errors);

      return errors.length > 0;
    };

    if (revealButton) {
      revealButton.onclick = (e) => {
        e.preventDefault();
        e.stopPropagation();

        const newType = input.type === "password" ? "text" : "password";
        const revealed = newType === "text";

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

  // Helper to update error summary visibility and focus (WCAG 3.3.6)
  const updateErrorSummary = (errorCount: number, focusOnErrors: boolean) => {
    if (errorCount > 0) {
      const fieldWord = errorCount === 1 ? "field" : "fields";
      errorSummaryText.textContent = `Please correct ${errorCount.toString()} ${fieldWord} with errors below`;
      errorSummary.classList.remove("hidden");
      if (focusOnErrors) {
        // Move focus to error summary for screen readers (WCAG 3.3.6)
        errorSummary.focus();
      }
    } else {
      errorSummary.classList.add("hidden");
    }
  };

  form.onsubmit = async (e) => {
    e.preventDefault();
    e.stopPropagation();

    const [newPassword] = fields.map((f) => f.getValue());

    const fieldErrors = fields.map(({ validate }) => validate());
    const hasErrors = fieldErrors.some((e) => e);
    const errorCount = fieldErrors.filter((e) => e).length;

    submitButton.disabled = hasErrors;
    updateErrorSummary(errorCount, true); // Focus on submit attempt

    if (hasErrors) return;

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
          const parsed = JSON.parse(body) as { data?: string[] };

          err = parsed.data?.[0] ?? body;
        } catch (_e) {
          // Ignore JSON parsing errors, use body as-is
        }

        throw new Error(`An error occurred: ${err}`);
      }

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

    const fieldErrors = fields.map(({ validate }) => validate());
    const hasErrors = fieldErrors.some((e) => e);
    const errorCount = fieldErrors.filter((e) => e).length;

    submitButton.disabled = hasErrors;
    updateErrorSummary(errorCount, false); // Don't focus on change
  };
};
