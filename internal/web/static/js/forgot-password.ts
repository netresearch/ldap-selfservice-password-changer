import { mustNotBeEmpty, isValidEmail } from "./validators.js";
import { initThemeToggle, initDensityToggle } from "./toggles.js";

export const init = () => {
  // Initialize theme and density toggles
  initThemeToggle();
  initDensityToggle();

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

  type Field = [string, ((v: string) => string)[]];

  const fieldsWithValidators = [["email", [mustNotBeEmpty("Email"), isValidEmail]]] satisfies Field[];

  const fields = fieldsWithValidators.map(([name, validators]) => {
    const f = form.querySelector<HTMLDivElement>(`#${name}`);
    if (!f) throw new Error(`Field "${name}" does not exist`);

    const inputContainer = f.querySelector<HTMLDivElement>('div[data-purpose="inputContainer"]');
    if (!inputContainer) throw new Error(`Input container for "${name}" does not exist`);

    const input = inputContainer.querySelector<HTMLInputElement>("input");
    if (!input) throw new Error(`Input for "${name}" does not exist`);

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
        .reduce<string[]>((acc, v) => {
          if (v.length > 0) acc.push(v);

          return acc;
        }, []);

      console.warn(`Validated "${name}": ${errors.length.toString()} error(s)`);

      setErrors(errors);

      return errors.length > 0;
    };

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

    const [email] = fields.map((f) => f.getValue());

    const hasErrors = fields.map(({ validate }) => validate()).some((e) => e);
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
          method: "request-password-reset",
          params: [email]
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

    const hasErrors = fields.map(({ validate }) => validate()).some((e) => e);
    submitButton.disabled = hasErrors;
  };
};
