import { mustNotBeEmpty, isValidEmail } from "./validators.js";
import { initThemeToggle, initDensityToggle } from "./toggles.js";
import { type FieldError, setFieldErrors, setSubmitError, updateErrorSummary } from "./error-utils.js";

export const init = () => {
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

  const submitButton = form.querySelector<HTMLButtonElement>(
    ":scope > div[data-purpose='submit'] > button[type='submit']"
  );
  if (!submitButton) throw new Error("Could not find submit button element");

  const submitErrorContainer = form.querySelector<HTMLDivElement>(
    ":scope > div[data-purpose='submit'] > div[data-purpose='errors']"
  );
  if (!submitErrorContainer) throw new Error("Could not find submit error container element");

  type Field = [string, string, ((v: string) => string)[]];

  const fieldsWithValidators = [["email", "Email Address", [mustNotBeEmpty("Email"), isValidEmail]]] satisfies Field[];

  const fields = fieldsWithValidators.map(([name, label, validators]) => {
    const f = form.querySelector<HTMLDivElement>(`#${name}`);
    if (!f) throw new Error(`Field "${name}" does not exist`);
    const inputContainer = f.querySelector<HTMLDivElement>('div[data-purpose="inputContainer"]');
    if (!inputContainer) throw new Error(`Input container for "${name}" does not exist`);
    const input = inputContainer.querySelector<HTMLInputElement>("input");
    if (!input) throw new Error(`Input for "${name}" does not exist`);
    const errorContainer = f.querySelector<HTMLDivElement>('div[data-purpose="errors"]');
    if (!errorContainer) throw new Error(`Error for "${name}" does not exist`);

    const getValue = () => input.value;
    const getErrors = (): string[] => validators.map((v) => v(getValue())).filter((msg) => msg.length > 0);
    const paint = (errors: string[]) => {
      setFieldErrors(errorContainer, inputContainer, input, errors);
    };

    const helpButton = f.querySelector<HTMLButtonElement>('button[data-purpose="help"]');
    const helpText = f.querySelector<HTMLDivElement>('div[data-purpose="helpText"]');
    if (helpButton && helpText) {
      helpButton.addEventListener("click", (e) => {
        e.preventDefault();
        const isExpanded = helpButton.getAttribute("aria-expanded") === "true";
        const newExpanded = !isExpanded;
        helpButton.setAttribute("aria-expanded", newExpanded.toString());
        helpText.classList.toggle("hidden", !newExpanded);
      });
    }

    return { name, label, input, getValue, getErrors, paint };
  });

  const touched = new Set<string>();

  const buildSummary = (): FieldError[] => {
    const out: FieldError[] = [];
    for (const f of fields) {
      const errs = f.getErrors();
      const first = errs[0];
      if (first) out.push({ fieldId: f.input.id, label: f.label, error: first });
    }
    return out;
  };

  const toggleFields = (enabled: boolean) => {
    [submitButton, ...fields.map(({ input }) => input)].forEach((el) => {
      el.disabled = !enabled;
    });
    submitButton.dataset["loading"] = (!enabled).toString();
    submitButton.setAttribute("aria-busy", (!enabled).toString());
  };

  form.addEventListener(
    "blur",
    (e) => {
      const target = e.target as HTMLElement;
      const f = fields.find((x) => x.input === target);
      if (!f) return;
      touched.add(f.name);
      f.paint(f.getErrors());
      if (!errorSummary.classList.contains("hidden")) {
        updateErrorSummary(errorSummary, errorSummaryText, buildSummary(), false);
      }
    },
    true
  );

  form.addEventListener("input", (e) => {
    const target = e.target as HTMLElement;
    const f = fields.find((x) => x.input === target);
    if (f && touched.has(f.name)) f.paint(f.getErrors());
    if (!errorSummary.classList.contains("hidden")) {
      updateErrorSummary(errorSummary, errorSummaryText, buildSummary(), false);
    }
  });

  const handleSubmit = async (e: SubmitEvent): Promise<void> => {
    e.preventDefault();

    for (const f of fields) {
      touched.add(f.name);
      f.paint(f.getErrors());
    }

    const summary = buildSummary();
    updateErrorSummary(errorSummary, errorSummaryText, summary, true);
    if (summary.length > 0) return;

    setSubmitError(submitErrorContainer, "");
    toggleFields(false);

    const [email] = fields.map((f) => f.getValue());

    try {
      const res = await fetch("/api/rpc", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ method: "request-password-reset", params: [email] })
      });

      const body = await res.text();
      if (!res.ok) {
        let err = body;
        try {
          const parsed = JSON.parse(body) as { data?: string[] };
          err = parsed.data?.[0] ?? body;
        } catch (_e) {
          // use raw body
        }
        throw new Error(err);
      }

      form.classList.add("hidden");
      successContainer.classList.remove("hidden");
    } catch (err) {
      setSubmitError(submitErrorContainer, (err as Error).message);
      toggleFields(true);
    }
  };

  form.addEventListener("submit", (e) => {
    void handleSubmit(e);
  });
};
