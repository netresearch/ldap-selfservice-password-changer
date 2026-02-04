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

/**
 * Updates error summary visibility and focus (WCAG 3.3.6)
 */
export function updateErrorSummary(
  errorSummary: HTMLElement,
  errorSummaryText: HTMLElement,
  errorCount: number,
  focusOnErrors: boolean
): void {
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

  if (errors.length > 0) {
    inputContainer.classList.add("!border-error-dark", "dark:!border-error-light");
    input.setAttribute("aria-invalid", "true");
  } else {
    inputContainer.classList.remove("!border-error-dark", "dark:!border-error-light");
    input.setAttribute("aria-invalid", "false");
  }

  for (const error of errors) {
    errorContainer.appendChild(createErrorElement(error));
  }
}
