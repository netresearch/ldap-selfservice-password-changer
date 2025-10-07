import {
  mustBeLongerThan,
  mustIncludeLowercase,
  mustIncludeNumbers,
  mustIncludeSymbols,
  mustIncludeUppercase,
  mustMatchNewPassword,
  mustNotBeEmpty,
  mustNotIncludeUsername,
  mustNotMatchCurrentPassword,
  toggleValidator
} from "./validators.js";

type Opts = {
  minLength: number;
  minNumbers: number;
  minSymbols: number;
  minUppercase: number;
  minLowercase: number;
  passwordCanIncludeUsername: boolean;
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
      themeToggle.dataset["theme"] = theme;

      // Update icon visibility
      themeLightIcon.classList.toggle("hidden", theme !== "light");
      themeDarkIcon.classList.toggle("hidden", theme !== "dark");
      themeAutoIcon.classList.toggle("hidden", theme !== "auto");

      // Update ARIA label to communicate current state and next action
      const nextTheme = theme === "auto" ? "light" : theme === "light" ? "dark" : "auto";
      const themeLabels = {
        light: "Light mode",
        dark: "Dark mode",
        auto: "Auto mode (follows system preference)"
      };
      themeToggle.setAttribute("aria-label", `Theme: ${themeLabels[theme]}. Click to switch to ${themeLabels[nextTheme]}`);

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
      const currentTheme = themeToggle.dataset["theme"] || "auto";
      const nextTheme = currentTheme === "auto" ? "light" : currentTheme === "light" ? "dark" : "auto";
      applyTheme(nextTheme);
    });
  }

  const successContainer = document.querySelector<HTMLFormElement>("div[data-purpose='successContainer']");
  if (!successContainer) throw new Error("Could not find success container element");

  const form = document.querySelector<HTMLFormElement>("#form");
  if (!form) throw new Error("Could not find form element");

  const submitButton = form.querySelector<HTMLButtonElement>("& > div[data-purpose='submit'] > button[type='submit']");
  if (!submitButton) throw new Error("Could not find submit button element");

  const submitErrorContainer = form.querySelector<HTMLDivElement>(
    "& > div[data-purpose='submit'] > div[data-purpose='errors']"
  );
  if (!submitErrorContainer) throw new Error("Could not find submit error container element");

  type Field = [string, string, ((v: string) => string)[]];

  const fieldsWithValidators = [
    ["username", "Username", [mustNotBeEmpty("Username")]],
    ["current", "Current Password", [mustNotBeEmpty("Current Password")]],
    [
      "new",
      "New Password",
      [
        mustNotBeEmpty("New Password"),
        mustBeLongerThan(opts.minLength, "New Password"),
        mustNotMatchCurrentPassword("New Password"),
        toggleValidator(mustNotIncludeUsername("New Password"), !opts.passwordCanIncludeUsername),
        mustIncludeNumbers(opts.minNumbers, "New Password"),
        mustIncludeSymbols(opts.minSymbols, "New Password"),
        mustIncludeUppercase(opts.minUppercase, "New Password"),
        mustIncludeLowercase(opts.minLowercase, "New Password")
      ]
    ],
    ["new2", "Password Confirmation", [mustNotBeEmpty("Password Confirmation"), mustMatchNewPassword("Password Confirmation")]]
  ] satisfies Field[];

  const fields = fieldsWithValidators.map(([name, _fieldLabel, validators]) => {
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
        inputContainer.classList.add("!border-red-700", "dark:!border-red-400");
        input.setAttribute("aria-invalid", "true");
      } else {
        inputContainer.classList.remove("!border-red-700", "dark:!border-red-400");
        input.setAttribute("aria-invalid", "false");
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

        // Update ARIA label to communicate current state
        revealButton.setAttribute("aria-label", revealed ? "Hide password" : "Show password");
        revealButton.setAttribute("aria-pressed", revealed.toString());
      };
    }

    return { input, errorContainer, getValue, validate };
  });

  const toggleFields = (enabled: boolean) => {
    [submitButton, ...fields.map(({ input }) => input)].forEach((el) => (el.disabled = !enabled));
    submitButton.dataset["loading"] = (!enabled).toString();
    submitButton.setAttribute("aria-busy", (!enabled).toString());
  };

  form.onsubmit = async (e) => {
    e.preventDefault();
    e.stopPropagation();

    const [username, oldPassword, newPassword] = fields.map((f) => f.getValue());

    const hasErrors = fields.map(({ validate }) => validate()).some((e) => e === true);
    submitButton.disabled = hasErrors;
    if (hasErrors) return;

    toggleFields(false);

    try {
      const res = await fetch("/api/rpc", {
        method: "POST",
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify({
          method: "change-password",
          params: [username, oldPassword, newPassword]
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

      form.style.display = "none";
      successContainer.style.display = "block";
    } catch (e) {
      console.error(e);

      submitErrorContainer.innerText = (e as Error).message;

      // Re-enable inputs but keep the submit button disabled,
      // since we know that this isn't going to work. After the validators
      // successfully re-run, it will enable the submit button again.
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
