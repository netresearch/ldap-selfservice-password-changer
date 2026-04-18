/**
 * Shared error handling utilities for form validation
 * WCAG 2.2 AAA compliance helpers
 */

// SVG error icon for inline use (WCAG 1.4.1 - not relying on color alone)
// SECURITY: This is a hardcoded constant, never contains user input
export const ERROR_ICON_SVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="inline-block h-3.5 w-3.5 shrink-0 mr-1" aria-hidden="true"><path fill-rule="evenodd" d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 5a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 5zm0 9a1 1 0 100-2 1 1 0 000 2z" clip-rule="evenodd" /></svg>`;

/**
 * Creates an error message element with icon
 * Uses safe DOM manipulation (no user data in innerHTML)
 */
export function createErrorElement(errorMessage: string): HTMLParagraphElement {
  const el = document.createElement("p");
  el.className = "flex items-start gap-1";

  // Create error icon using template element for safe SVG parsing
  // SECURITY: ERROR_ICON_SVG is a hardcoded constant defined above, not user input
  const template = document.createElement("template");
  template.innerHTML = ERROR_ICON_SVG;
  const icon = template.content.firstChild;
  if (icon) el.appendChild(icon);

  // Create text span with textContent which safely escapes all HTML
  const textSpan = document.createElement("span");
  textSpan.textContent = errorMessage;
  el.appendChild(textSpan);

  return el;
}

/** One entry in the error summary — a field and its first error message. */
export interface FieldError {
  fieldId: string; // id of the <input>, used as the anchor target
  label: string;
  error: string;
}

/**
 * Updates error summary visibility, content and focus (WCAG 3.3.6).
 * When errors exist, renders a linked list (<ol> of anchor links) so users
 * can jump directly to the broken field.
 */
export function updateErrorSummary(
  errorSummary: HTMLElement,
  errorSummaryText: HTMLElement,
  fieldErrors: FieldError[],
  focusOnErrors: boolean
): void {
  const errorCount = fieldErrors.length;

  if (errorCount === 0) {
    errorSummary.classList.add("hidden");
    return;
  }

  const fieldWord = errorCount === 1 ? "field" : "fields";
  errorSummaryText.textContent = `Please correct ${errorCount.toString()} ${fieldWord} below`;

  let list = errorSummary.querySelector<HTMLOListElement>("ol[data-purpose='errorList']");
  if (!list) {
    list = document.createElement("ol");
    list.dataset["purpose"] = "errorList";
    list.className = "mt-2 ml-5 list-decimal space-y-1 text-sm text-error-dark dark:text-error-light";
    errorSummary.appendChild(list);
  }
  while (list.firstChild) list.removeChild(list.firstChild);

  for (const fe of fieldErrors) {
    const li = document.createElement("li");
    const a = document.createElement("a");
    a.href = `#${fe.fieldId}`;
    a.className = "underline hover:no-underline focus:ring-2 focus:ring-error-dark dark:focus:ring-error-light rounded";
    a.textContent = `${fe.label}: ${fe.error}`;
    a.addEventListener("click", (e) => {
      e.preventDefault();
      const target = document.getElementById(fe.fieldId);
      if (target) {
        target.focus();
        // Respect prefers-reduced-motion (WCAG 2.3.3).
        const prefersReducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
        target.scrollIntoView({ block: "center", behavior: prefersReducedMotion ? "auto" : "smooth" });
      }
    });
    li.appendChild(a);
    list.appendChild(li);
  }

  errorSummary.classList.remove("hidden");
  if (focusOnErrors) errorSummary.focus();
}

/**
 * Render a server/submit error using the shared error element, so the visual
 * matches the inline field errors (icon + text).
 */
export function setSubmitError(container: HTMLElement, message: string): void {
  while (container.firstChild) container.removeChild(container.firstChild);
  if (message) container.appendChild(createErrorElement(message));
}

/**
 * Toggle red border + aria-invalid without rendering inline error text.
 * Used when error text is surfaced elsewhere (e.g. the live password-policy
 * checklist takes over the new_password text channel).
 */
export function setFieldInvalidStyle(inputContainer: HTMLElement, input: HTMLInputElement, invalid: boolean): void {
  if (invalid) {
    inputContainer.classList.add("!border-error-dark", "dark:!border-error-light");
    input.setAttribute("aria-invalid", "true");
  } else {
    inputContainer.classList.remove("!border-error-dark", "dark:!border-error-light");
    input.setAttribute("aria-invalid", "false");
  }
}

/**
 * Sets errors on a field's error container with proper styling
 */
export function setFieldErrors(
  errorContainer: HTMLElement,
  inputContainer: HTMLElement,
  input: HTMLInputElement,
  errors: string[]
): void {
  // Clear existing errors using safe DOM method
  while (errorContainer.firstChild) {
    errorContainer.removeChild(errorContainer.firstChild);
  }

  setFieldInvalidStyle(inputContainer, input, errors.length > 0);

  for (const error of errors) {
    errorContainer.appendChild(createErrorElement(error));
  }
}
